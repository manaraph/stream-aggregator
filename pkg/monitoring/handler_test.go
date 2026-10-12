package monitoring

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/protobuf/proto"
)

func TestHandlerExportsLatestMetricsAndReportTimes(t *testing.T) {
	reportedAt := time.Unix(123, 0)
	metrics := &streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{
			Received: proto.Uint64(11), Processed: proto.Uint64(10), Dropped: proto.Uint64(1),
			Used: proto.Uint32(2), Capacity: proto.Uint32(8), MaxUsed: proto.Uint32(4), Utilization: proto.Float64(25),
			DbBatchDurationMilliseconds: proto.Float64(2.5), DbBatchFailures: proto.Uint64(3),
			DbBatchSize: proto.Uint32(5), DbRowsCommitted: proto.Uint64(10),
		},
		Throughput: &streamv1.ThroughputMetrics{
			PublishRate: proto.Float64(100), IngestionRate: proto.Float64(99),
			GrpcRate: proto.Float64(98), WebsocketRate: proto.Float64(97),
		},
		Latency: &streamv1.LatencyMetrics{AverageMs: proto.Uint32(2), P95Ms: proto.Uint32(3), MaxMs: proto.Uint32(4)},
		Grpc:    &streamv1.ConnectionMetrics{Connected: proto.Bool(true), Reconnects: proto.Uint32(1), Errors: proto.Uint32(2)},
		Broker:  &streamv1.ConnectionMetrics{Connected: proto.Bool(false), Reconnects: proto.Uint32(3), Errors: proto.Uint32(4)},
		Runtime: &streamv1.RuntimeMetrics{
			UptimeSeconds: proto.Uint64(12), Goroutines: proto.Uint32(13), MemoryBytes: proto.Uint64(14),
			WebsocketClients: proto.Uint32(15), BackpressureLevel: proto.Uint32(16), WebsocketDropped: proto.Uint64(17),
		},
	}
	handler := Handler(func() *streamv1.IngestMetricsRequest { return metrics }, func() map[string]time.Time {
		return map[string]time.Time{"ingestion": reportedAt}
	})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))

	for _, want := range []string{
		"stream_aggregator_service_last_report_timestamp_seconds{service=\"ingestion\"} 123.000000",
		"stream_aggregator_queue_received_total 11", "stream_aggregator_queue_processed_total 10",
		"stream_aggregator_queue_dropped_total 1", "stream_aggregator_queue_utilization_percent 25",
		"stream_aggregator_db_batch_duration_milliseconds 2.5", "stream_aggregator_db_batch_failures_total 3",
		"stream_aggregator_db_batch_size 5", "stream_aggregator_db_rows_committed_total 10",
		"stream_aggregator_publish_rate_per_second 100", "stream_aggregator_ingestion_rate_per_second 99",
		"stream_aggregator_grpc_rate_per_second 98", "stream_aggregator_websocket_delivery_rate_per_second 97",
		"stream_aggregator_latency_average_milliseconds 2", "stream_aggregator_latency_p95_milliseconds 3",
		"stream_aggregator_latency_max_milliseconds 4", "stream_aggregator_grpc_connected 1",
		"stream_aggregator_broker_connected 0", "stream_aggregator_grpc_reconnects_total 1",
		"stream_aggregator_broker_errors_total 4", "stream_aggregator_uptime_seconds 12",
		"stream_aggregator_goroutines 13", "stream_aggregator_memory_bytes 14",
		"stream_aggregator_websocket_clients 15", "stream_aggregator_backpressure_level 16",
		"stream_aggregator_websocket_dropped_total 17",
	} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("metrics response missing %q", want)
		}
	}
}

func TestHandlerHandlesNilSnapshotAndNoReports(t *testing.T) {
	handler := Handler(func() *streamv1.IngestMetricsRequest { return nil })
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty metrics body, got %q", recorder.Body.String())
	}
}
