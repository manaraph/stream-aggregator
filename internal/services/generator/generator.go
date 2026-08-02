package generator

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
)

var interval time.Duration

type Publisher struct {
	B          broker.Broker
	GRPC       *grpc.ClientConn
	M          MetricsStreamClient
	published  uint64
	publishErr uint32
}

type MetricsStreamClient interface {
	Send(*streamv1.IngestMetricsRequest) error
}

func (p *Publisher) SendEvent(e domain.Sensor) error {
	data, _ := json.Marshal(e)
	if err := p.B.Publish("sensors/temperature", data); err != nil {
		atomic.AddUint32(&p.publishErr, 1)
		return err
	}
	atomic.AddUint64(&p.published, 1)
	return nil
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	metricsTicker := time.NewTicker(5 * time.Second)
	defer metricsTicker.Stop()
	var previousPublished uint64

	for {
		select {
		case <-ctx.Done():
			log.Println("Publisher stopping: context cancelled")
			return
		case <-ticker.C:
			event := domain.Sensor{
				Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
				Value:     10 + rand.Float64()*20,
				Timestamp: time.Now().UTC(),
			}

			if err := p.SendEvent(event); err != nil {
				log.Printf("Failed to send event: %v", err)
			}
		case <-metricsTicker.C:
			previousPublished = p.reportMetrics(previousPublished, 5*time.Second)
		}
	}
}

func (p *Publisher) reportMetrics(previousPublished uint64, duration time.Duration) uint64 {
	published := atomic.LoadUint64(&p.published)
	if p.M == nil {
		return published
	}

	metrics := &streamv1.IngestMetricsRequest{
		Throughput: &streamv1.ThroughputMetrics{PublishRate: float64(published-previousPublished) / duration.Seconds()},
		Broker:     &streamv1.ConnectionMetrics{Connected: p.B != nil, Errors: atomic.LoadUint32(&p.publishErr)},
	}
	if err := p.M.Send(metrics); err != nil {
		log.Println("gRPC metrics send failed:", err)
	}
	return published
}

func (p *Publisher) Close() error {
	if p.B != nil {
		_ = p.B.Close()
	}
	if p.GRPC != nil {
		return p.GRPC.Close()
	}
	return nil
}

func init() {
	rateStr := os.Getenv("PUBLISH_RATE")
	if rateStr == "" {
		rateStr = "1"
	}
	rate := 1 // events/sec
	if v, err := strconv.Atoi(rateStr); err == nil && v > 0 {
		rate = v
	}

	interval = time.Second / time.Duration(rate)
	log.Printf("Publishing at %d events/sec (interval %v)", rate, interval)
}

func NewPublisher() (*Publisher, error) {
	clientId := os.Getenv("GENERATOR_ID")
	if clientId == "" {
		return nil, errors.New("GENERATOR_ID not defined")
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		return nil, err
	}

	_, conn, err := grpcapi.ConnectGateway()
	if err != nil {
		log.Printf("metrics client unavailable: %v", err)
		return &Publisher{B: mclient}, nil
	}
	metricsStream, err := streamv1.NewMetricsServiceClient(conn).IngestMetrics(context.Background())
	if err != nil {
		_ = conn.Close()
		log.Printf("metrics stream unavailable: %v", err)
		return &Publisher{B: mclient}, nil
	}

	return &Publisher{B: mclient, GRPC: conn, M: metricsStream}, nil
}
