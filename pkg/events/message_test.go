package events

import (
	"encoding/json"
	"strings"
	"testing"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestNewMetricsMessageUsesCamelCaseJSON(t *testing.T) {
	message := NewMetricsMessage(&streamv1.IngestMetricsRequest{
		Queue:      &streamv1.QueueMetrics{MaxUsed: proto.Uint32(3)},
		Throughput: &streamv1.ThroughputMetrics{WebsocketRate: proto.Float64(2.5)},
		Runtime:    &streamv1.RuntimeMetrics{UptimeSeconds: proto.Uint64(10), BackpressureLevel: proto.Uint32(4)},
	})

	payload, err := json.Marshal(message)
	assert.NoError(t, err)
	json := string(payload)
	assert.True(t, strings.Contains(json, `"maxUsed":3`))
	assert.True(t, strings.Contains(json, `"websocketRate":2.5`))
	assert.True(t, strings.Contains(json, `"uptimeSeconds":10`))
	assert.True(t, strings.Contains(json, `"backpressureLevel":4`))
	assert.False(t, strings.Contains(json, "max_used"))
	assert.False(t, strings.Contains(json, "websocket_rate"))
}

func TestNewMetricsMessageConvertsAllMetricSections(t *testing.T) {
	message := NewMetricsMessage(&streamv1.IngestMetricsRequest{
		Queue:      &streamv1.QueueMetrics{Processed: proto.Uint64(10), Dropped: proto.Uint64(2), Used: proto.Uint32(5), Capacity: proto.Uint32(8), MaxUsed: proto.Uint32(7), Utilization: proto.Float64(0.75)},
		Throughput: &streamv1.ThroughputMetrics{PublishRate: proto.Float64(3.2), IngestionRate: proto.Float64(2.8), GrpcRate: proto.Float64(1.4), WebsocketRate: proto.Float64(0.9)},
		Latency:    &streamv1.LatencyMetrics{AverageMs: proto.Uint32(12), P95Ms: proto.Uint32(25), MaxMs: proto.Uint32(40)},
		Grpc:       &streamv1.ConnectionMetrics{Connected: proto.Bool(true), Reconnects: proto.Uint32(1), Errors: proto.Uint32(2)},
		Broker:     &streamv1.ConnectionMetrics{Connected: proto.Bool(false), Reconnects: proto.Uint32(3), Errors: proto.Uint32(4)},
		Runtime:    &streamv1.RuntimeMetrics{UptimeSeconds: proto.Uint64(120), Goroutines: proto.Uint32(9), MemoryBytes: proto.Uint64(2048), WebsocketClients: proto.Uint32(6), BackpressureLevel: proto.Uint32(1)},
	})

	metrics, ok := message.Data.(MetricsData)
	assert.True(t, ok)
	assert.NotNil(t, metrics.Queue)
	assert.NotNil(t, metrics.Throughput)
	assert.NotNil(t, metrics.Latency)
	assert.NotNil(t, metrics.Grpc)
	assert.NotNil(t, metrics.Broker)
	assert.NotNil(t, metrics.Runtime)
}

func TestNewMetricsMessageHandlesEmptyPayload(t *testing.T) {
	message := NewMetricsMessage(&streamv1.IngestMetricsRequest{})
	metrics, ok := message.Data.(MetricsData)
	assert.True(t, ok)
	assert.Nil(t, metrics.Queue)
	assert.Nil(t, metrics.Throughput)
	assert.Nil(t, metrics.Latency)
	assert.Nil(t, metrics.Grpc)
	assert.Nil(t, metrics.Broker)
	assert.Nil(t, metrics.Runtime)
}
