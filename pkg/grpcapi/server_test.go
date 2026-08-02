package grpcapi

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/manaraph/stream-aggregator/pkg/events"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockDispatcher struct {
	events chan events.Message
}

func (m *mockDispatcher) Publish(event events.Message) {
	m.events <- event
}

// Mock gRPC Stream
type mockIngestStream struct {
	streamv1.SensorService_IngestSensorServer
	ctx    context.Context
	reqCh  chan *streamv1.IngestSensorRequest
	closed bool
}

type mockMetricsStream struct {
	streamv1.SensorService_StreamMetricsServer
	reqCh chan *streamv1.StreamMetricsRequest
}

func (m *mockMetricsStream) Recv() (*streamv1.StreamMetricsRequest, error) {
	req, ok := <-m.reqCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (m *mockIngestStream) Recv() (*streamv1.IngestSensorRequest, error) {
	req, ok := <-m.reqCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func TestStreamMetricsPublishesTypedEvent(t *testing.T) {
	req := &streamv1.StreamMetricsRequest{Processed: 42, Rate: 12.5}
	dispatcher := &mockDispatcher{events: make(chan events.Message, 1)}
	stream := &mockMetricsStream{reqCh: make(chan *streamv1.StreamMetricsRequest, 1)}
	server := &Server{Dispatcher: dispatcher}

	go func() {
		_ = server.StreamMetrics(stream)
	}()

	stream.reqCh <- req
	close(stream.reqCh)

	select {
	case event := <-dispatcher.events:
		assert.Equal(t, "metrics", event.Type)
		got, ok := event.Data.(*streamv1.StreamMetricsRequest)
		assert.True(t, ok, "received unexpected event data")
		if diff := cmp.Diff(req, got, protocmp.Transform()); diff != "" {
			t.Errorf("StreamMetricsRequest mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: metrics event was not dispatched")
	}
}

func (m *mockIngestStream) Context() context.Context {
	return m.ctx
}

// Tests
func TestIngestSensor(t *testing.T) {
	now := timestamppb.Now()
	expectedValue := 25.060459624734243

	req := &streamv1.IngestSensorRequest{
		Sensor:    "sensor-D",
		Value:     expectedValue,
		Timestamp: now,
	}

	dispatcher := &mockDispatcher{
		events: make(chan events.Message, 1),
	}

	stream := &mockIngestStream{
		ctx:   context.Background(),
		reqCh: make(chan *streamv1.IngestSensorRequest, 1),
	}

	server := &Server{
		Dispatcher: dispatcher,
	}

	go func() {
		_ = server.IngestSensor(stream)
	}()

	stream.reqCh <- req
	close(stream.reqCh) // triggers EOF

	select {
	case event := <-dispatcher.events:
		assert.Equal(t, "sensor", event.Type)
		got, ok := event.Data.(*streamv1.IngestSensorRequest)
		assert.True(t, ok, "received unexpected event data")

		if diff := cmp.Diff(req, got, protocmp.Transform()); diff != "" {
			t.Errorf("IngestSensorRequest mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout: hub did not receive event")
	}
}
