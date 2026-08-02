package grpcapi

import (
	"testing"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestWebSocketDispatcherMergesPartialMetricUpdates(t *testing.T) {
	dispatcher := NewWebSocketDispatcher(ws.NewHub())
	dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{
			Processed: proto.Uint64(7),
			Capacity:  proto.Uint32(100),
		},
	})
	dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{Processed: proto.Uint64(0)},
	})

	assert.Equal(t, uint64(0), dispatcher.metrics.GetQueue().GetProcessed())
	assert.Equal(t, uint32(100), dispatcher.metrics.GetQueue().GetCapacity())
}
