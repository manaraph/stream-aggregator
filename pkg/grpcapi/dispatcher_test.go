package grpcapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
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

func TestWebSocketDispatcherIgnoresNilMetricUpdates(t *testing.T) {
	dispatcher := NewWebSocketDispatcher(ws.NewHub())
	dispatcher.PublishMetrics(nil)

	assert.Nil(t, dispatcher.metrics)
}

func TestWebSocketDispatcherReturnsMetricSnapshotAndReportTimes(t *testing.T) {
	dispatcher := NewWebSocketDispatcher(ws.NewHub())
	dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{Processed: proto.Uint64(7)},
	})

	snapshot := dispatcher.MetricsSnapshot()
	if snapshot == nil || snapshot.GetQueue().GetProcessed() != 7 {
		t.Fatalf("unexpected metrics snapshot: %v", snapshot)
	}

	if got := dispatcher.LastReports(); len(got) != 1 || got["ingestion"].IsZero() {
		t.Fatalf("unexpected report timestamps: %v", got)
	}
}

func TestWebSocketDispatcherPublishesSnapshotAfterMerge(t *testing.T) {
	dispatcher := NewWebSocketDispatcher(ws.NewHub())
	dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{Processed: proto.Uint64(5)},
	})
	dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{Capacity: proto.Uint32(100)},
	})

	assert.NotNil(t, dispatcher.metrics)
	assert.Equal(t, uint64(5), dispatcher.metrics.GetQueue().GetProcessed())
	assert.Equal(t, uint32(100), dispatcher.metrics.GetQueue().GetCapacity())
}
