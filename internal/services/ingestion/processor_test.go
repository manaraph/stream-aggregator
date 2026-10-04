package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	ingestionmocks "github.com/manaraph/stream-aggregator/internal/services/ingestion/mocks"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"
)

func TestProcessor_FullLifecycle(t *testing.T) {
	fakeBroker := broker.NewFakeBroker()
	mockStream := ingestionmocks.NewMockSensorStreamClient(t)

	p := &Processor{
		B:          fakeBroker,
		store:      &testBatchStore{},
		S:          mockStream,
		wg:         sync.WaitGroup{},
		eventQueue: make(chan queuedReading, 10),
	}

	mockStream.EXPECT().Send(mock.Anything).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Run(ctx)
	assert.NoError(t, err)

	// Simulate an MQTT message arriving.
	sensor := domain.Sensor{Sensor: "test-device", Value: 99.9, Timestamp: time.Now()}
	payload, _ := json.Marshal(sensor)
	mockMsg := &broker.MockMessage{PayloadData: payload, TopicData: "sensors/temperature"}

	p.HandleMessage(nil, mockMsg)

	assert.NoError(t, p.Close(ctx))
	assert.Equal(t, uint64(1), atomic.LoadUint64(&p.processed))
	assert.True(t, mockMsg.Acked)

	// Verify mock was called before Close finished
	mockStream.AssertExpectations(t)
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	p := &Processor{processed: 0}

	// Send invalid data
	mockMsg := &broker.MockMessage{PayloadData: []byte("invalid-json{")}

	p.HandleMessage(nil, mockMsg)

	assert.Equal(t, uint64(0), atomic.LoadUint64(&p.processed), "Invalid JSON should not be queued")
}

func TestProcessor_Close_Timeout(t *testing.T) {
	p := &Processor{
		eventQueue: make(chan queuedReading, 1),
	}
	p.wg.Add(1) // Simulate a stuck worker that never calls Done()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := p.Close(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestForwardEvent_Errors(t *testing.T) {
	t.Run("Nil StreamClient", func(t *testing.T) {
		p := &Processor{S: nil}
		p.ForwardEvent(domain.Sensor{})
	})

	t.Run("gRPC Send Failed", func(t *testing.T) {
		mockS := ingestionmocks.NewMockSensorStreamClient(t)
		mockS.EXPECT().Send(mock.Anything).Return(errors.New("connection lost"))

		p := &Processor{S: mockS}
		p.ForwardEvent(domain.Sensor{Sensor: "test"})

		mockS.AssertExpectations(t)
	})
}

func TestReportQueueStatusForwardsMetrics(t *testing.T) {
	metricsStream := ingestionmocks.NewMockMetricsStreamClient(t)
	p := &Processor{
		eventQueue: make(chan queuedReading, 4),
		M:          metricsStream,
		processed:  9,
		dropped:    2,
	}
	p.eventQueue <- queuedReading{}
	p.eventQueue <- queuedReading{}

	metricsStream.EXPECT().Send(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{
			Processed:   proto.Uint64(9),
			Dropped:     proto.Uint64(2),
			Used:        proto.Uint32(2),
			Capacity:    proto.Uint32(4),
			MaxUsed:     proto.Uint32(0),
			Utilization: proto.Float64(50),
		},
		Throughput: &streamv1.ThroughputMetrics{IngestionRate: proto.Float64(1)},
		Grpc:       &streamv1.ConnectionMetrics{Connected: proto.Bool(false), Errors: proto.Uint32(0)},
		Broker:     &streamv1.ConnectionMetrics{Connected: proto.Bool(false)},
	}).Return(nil).Once()

	processed := p.reportQueueStatus(4, 5*time.Second)
	assert.Equal(t, uint64(9), processed)
	metricsStream.AssertExpectations(t)
}
