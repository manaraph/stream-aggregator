package grpcapi

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock Hub
type mockHub struct {
	Events chan interface{}
}

func (m *mockHub) BroadcastEvent(e interface{}) {
	m.Events <- e
}

// Mock gRPC Stream
type mockIngestStream struct {
	streamv1.SensorService_IngestSensorServer
	ctx    context.Context
	reqCh  chan *streamv1.IngestSensorRequest
	closed bool
}

func (m *mockIngestStream) Recv() (*streamv1.IngestSensorRequest, error) {
	req, ok := <-m.reqCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
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

	mockHub := &mockHub{
		Events: make(chan interface{}, 1),
	}

	stream := &mockIngestStream{
		ctx:   context.Background(),
		reqCh: make(chan *streamv1.IngestSensorRequest, 1),
	}

	server := &Server{
		Hub: mockHub,
	}

	go func() {
		_ = server.IngestSensor(stream)
	}()

	stream.reqCh <- req
	close(stream.reqCh) // triggers EOF

	select {
	case raw := <-mockHub.Events:
		got, ok := raw.(*streamv1.IngestSensorRequest)
		assert.True(t, ok, "received unexpected type from hub")

		if diff := cmp.Diff(req, got, protocmp.Transform()); diff != "" {
			t.Errorf("IngestSensorRequest mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout: hub did not receive event")
	}
}
