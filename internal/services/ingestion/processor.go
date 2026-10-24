package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
)

type SensorStreamClient interface {
	Send(*streamv1.IngestSensorRequest) error
}

type MetricsStreamClient interface {
	Send(*streamv1.IngestMetricsRequest) error
}

type sensorStore interface {
	InsertBatch(context.Context, []domain.Sensor) error
	Close()
}

type Processor struct {
	B                 broker.Broker
	store             sensorStore
	GRPC              *grpc.ClientConn
	S                 SensorStreamClient
	M                 MetricsStreamClient
	eventQueue        chan queuedReading
	processed         uint64
	received          uint64
	dropped           uint64
	maxUsed           uint32
	grpcErrors        uint32
	dbBatchDurationNS uint64
	dbBatchFailures   uint64
	dbBatchSize       uint32
	dbRowsCommitted   uint64
	wg                sync.WaitGroup
	callbackWG        sync.WaitGroup
	callbackMu        sync.RWMutex
	closing           bool
	cancel            context.CancelFunc
	ctx               context.Context
	writerCtx         context.Context
	writerStop        context.CancelFunc
}

type queuedReading struct {
	reading domain.Sensor
	message mqtt.Message
}

func (p *Processor) Run(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	p.initPipeline()
	if err := p.B.Subscribe("sensors/#", p.HandleMessage); err != nil {
		return fmt.Errorf("subscribe to sensors/#: %w", err)
	}
	log.Println("Subscribed to MQTT topic sensors/#")
	return nil
}

func (p *Processor) HandleMessage(c mqtt.Client, m mqtt.Message) {
	p.callbackMu.RLock()
	if p.closing {
		p.callbackMu.RUnlock()
		return
	}
	p.callbackWG.Add(1)
	p.callbackMu.RUnlock()
	defer p.callbackWG.Done()
	received := atomic.AddUint64(&p.received, 1)
	if received == 1 {
		log.Printf("Received first MQTT message on topic %s", m.Topic())
	}

	var e domain.Sensor
	if err := json.Unmarshal(m.Payload(), &e); err != nil {
		log.Println("Invalid event:", err)
		m.Ack()
		return
	}
	if e.EventID == "" {
		sum := sha256.Sum256(append(append([]byte(m.Topic()), 0), m.Payload()...))
		e.EventID = hex.EncodeToString(sum[:])
	}
	if e.MeasurementType == "" {
		e.MeasurementType = "temperature"
	}
	if e.Unit == "" {
		e.Unit = "C"
	}
	p.enqueueEvent(e, m)
}

func (p *Processor) ForwardEvent(data domain.Sensor) {
	if p.S == nil {
		log.Println("ERROR: StreamClient is nil!")
		return
	}

	err := p.S.Send(&streamv1.IngestSensorRequest{
		Sensor:          data.Sensor,
		Value:           data.Value,
		Timestamp:       timestamppb.New(data.Timestamp),
		EventId:         data.EventID,
		MeasurementType: data.MeasurementType,
		Unit:            data.Unit,
	})
	if err != nil {
		log.Println("gRPC send failed:", err)
		atomic.AddUint32(&p.grpcErrors, 1)
	}
}

func (p *Processor) ForwardMetrics(metrics *streamv1.IngestMetricsRequest) {
	if p.M == nil {
		return
	}
	if err := p.M.Send(metrics); err != nil {
		log.Println("gRPC metrics send failed:", err)
		atomic.AddUint32(&p.grpcErrors, 1)
	}
}

func (p *Processor) Close(ctx context.Context) error {
	log.Println("Shutting down processor...")

	if p.B != nil {
		p.B.Close()
	}
	p.callbackMu.Lock()
	p.closing = true
	p.callbackMu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	callbacksDone := make(chan struct{})
	go func() {
		p.callbackWG.Wait()
		close(callbacksDone)
	}()
	select {
	case <-callbacksDone:
	case <-ctx.Done():
		if p.writerStop != nil {
			p.writerStop()
		}
		return fmt.Errorf("shutdown timed out waiting for MQTT callbacks: %w", ctx.Err())
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Workers drained successfully")
	case <-ctx.Done():
		if p.writerStop != nil {
			p.writerStop()
		}
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
	if p.writerStop != nil {
		p.writerStop()
	}
	if p.store != nil {
		p.store.Close()
	}

	if p.GRPC != nil {
		log.Println("Closing gRPC connection")
		return p.GRPC.Close()
	}

	return nil
}
