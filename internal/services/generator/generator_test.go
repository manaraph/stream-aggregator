package generator

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type stubMetricsClient struct {
	requests []*streamv1.IngestMetricsRequest
}

func (s *stubMetricsClient) Send(req *streamv1.IngestMetricsRequest) error {
	s.requests = append(s.requests, req)
	return nil
}

func TestPublisher_Run(t *testing.T) {
	interval = 10 * time.Millisecond
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go pub.Run(ctx)

	for i := 0; i < 3; i++ {
		select {
		case msg := <-fake.Messages:
			var s domain.Sensor
			err := json.Unmarshal(msg, &s)
			assert.NoError(t, err)
			assert.Contains(t, s.Sensor, "sensor-")
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Failed to receive event %d", i)
		}
	}
}

func TestPublisher_Cancellation(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	pub.Run(ctx)
	duration := time.Since(start)

	assert.True(t, duration >= 50*time.Millisecond)
	assert.True(t, duration < 100*time.Millisecond, "Run took too long to exit")
}

func TestSendEvent(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	event := domain.Sensor{
		Sensor:    "sensor-A",
		Value:     42.0,
		Timestamp: time.Now().UTC(),
	}

	err := pub.SendEvent(event)
	assert.NoError(t, err)

	select {
	case msg := <-fake.Messages:
		var got domain.Sensor
		err := json.Unmarshal(msg, &got)
		assert.NoError(t, err)
		assert.Equal(t, "sensor-A", got.Sensor)
		assert.Equal(t, 42.0, got.Value)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout: Broker never received the message")
	}
}

func TestPublisher_ReportMetrics(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake, published: 5, publishErr: 2}

	assert.Equal(t, uint64(5), pub.reportMetrics(2, 5*time.Second))

	client := &stubMetricsClient{}
	pub.M = client
	assert.Equal(t, uint64(5), pub.reportMetrics(2, 5*time.Second))
	if assert.Len(t, client.requests, 1) {
		request := client.requests[0]
		assert.NotNil(t, request.GetThroughput())
		assert.NotNil(t, request.GetBroker())
		assert.Equal(t, float64(3)/5, request.GetThroughput().GetPublishRate())
		assert.True(t, request.GetBroker().GetConnected())
		assert.Equal(t, uint32(2), request.GetBroker().GetErrors())
	}
}

func TestPublisher_Close(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}
	assert.NoError(t, pub.Close())

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer lis.Close()

	s := grpc.NewServer()
	go s.Serve(lis)
	defer s.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	pubWithConn := &Publisher{GRPC: conn}
	assert.NoError(t, pubWithConn.Close())
}

func TestNewPublisherFromEnv(t *testing.T) {
	t.Run("Fails when env vars are missing", func(t *testing.T) {
		os.Unsetenv("GENERATOR_ID")
		os.Unsetenv("MQTT_BROKER")

		pub, err := NewPublisher()
		assert.Nil(t, pub)
		assert.Error(t, err)
	})

	t.Run("Fails with invalid broker URL", func(t *testing.T) {
		os.Setenv("GENERATOR_ID", "test-gen")
		os.Setenv("MQTT_BROKER", "tcp://invalid-address:9999")
		defer os.Clearenv()

		pub, err := NewPublisher()
		assert.Nil(t, pub)
		assert.Error(t, err)
	})
}
