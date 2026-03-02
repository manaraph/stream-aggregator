package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StreamClient interface {
	Send(*streamv1.IngestSensorRequest) error
}

type Processor struct {
	B          broker.Broker
	GRPC       *grpc.ClientConn
	S          StreamClient
	eventQueue chan domain.Sensor
	processed  uint64
	dropped    uint64
	WG         sync.WaitGroup
	cancel     context.CancelFunc
}

func (p *Processor) Run(ctx context.Context) error {
	_, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.initPipeline()
	return p.B.Subscribe("sensors/#", p.HandleMessage)
}

func (p *Processor) HandleMessage(c mqtt.Client, m mqtt.Message) {
	var e domain.Sensor
	if err := json.Unmarshal(m.Payload(), &e); err != nil {
		log.Println("Invalid event:", err)
		return
	}
	p.EnqueueEvent(e)
}

func (p *Processor) ForwardEvent(data domain.Sensor) {
	if p.S == nil {
		log.Println("ERROR: StreamClient is nil!")
		return
	}

	err := p.S.Send(&streamv1.IngestSensorRequest{
		Sensor:    data.Sensor,
		Value:     data.Value,
		Timestamp: timestamppb.New(data.Timestamp),
	})

	if err != nil {
		log.Println("gRPC send failed:", err)
	}
}

func (p *Processor) Close(ctx context.Context) error {
	log.Println("Shutting down processor...")

	if p.cancel != nil {
		p.cancel()
	}

	if p.B != nil {
		p.B.Close()
	}

	close(p.eventQueue)

	done := make(chan struct{})
	go func() {
		p.WG.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Workers drained successfully")
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}

	if p.GRPC != nil {
		log.Println("Closing gRPC connection")
		return p.GRPC.Close()
	}

	return nil
}
