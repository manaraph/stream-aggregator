// Package monitoring exposes the gateway's latest service metrics in the
// Prometheus text exposition format.
package monitoring

import (
	"fmt"
	"net/http"
	"time"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
)

type (
	snapshotter func() *streamv1.IngestMetricsRequest
	reportTimes func() map[string]time.Time
)

// Handler serves the most recently received metrics snapshot. Missing values
// are omitted until their owning service reports them.
func Handler(snapshot snapshotter, reports ...reportTimes) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m := snapshot()
		if len(reports) > 0 {
			for service, reportedAt := range reports[0]() {
				_, _ = fmt.Fprintf(
					w,
					"stream_aggregator_service_last_report_timestamp_seconds{service=%q} %f\n",
					service, float64(reportedAt.UnixNano())/float64(time.Second),
				)
			}
		}
		if m == nil {
			return
		}
		write := func(name string, value any) {
			_, _ = fmt.Fprintf(w, "stream_aggregator_%s %v\n", name, value)
		}
		if q := m.GetQueue(); q != nil {
			write("queue_received_total", q.GetReceived())
			write("queue_processed_total", q.GetProcessed())
			write("queue_dropped_total", q.GetDropped())
			write("queue_used", q.GetUsed())
			write("queue_capacity", q.GetCapacity())
			write("queue_high_water_mark", q.GetMaxUsed())
			write("queue_utilization_percent", q.GetUtilization())
			write("db_batch_duration_milliseconds", q.GetDbBatchDurationMilliseconds())
			write("db_batch_failures_total", q.GetDbBatchFailures())
			write("db_batch_size", q.GetDbBatchSize())
			write("db_rows_committed_total", q.GetDbRowsCommitted())
		}
		if t := m.GetThroughput(); t != nil {
			write("publish_rate_per_second", t.GetPublishRate())
			write("ingestion_rate_per_second", t.GetIngestionRate())
			write("grpc_rate_per_second", t.GetGrpcRate())
			write("websocket_delivery_rate_per_second", t.GetWebsocketRate())
		}
		if l := m.GetLatency(); l != nil {
			write("latency_average_milliseconds", l.GetAverageMs())
			write("latency_p95_milliseconds", l.GetP95Ms())
			write("latency_max_milliseconds", l.GetMaxMs())
		}
		if c := m.GetGrpc(); c != nil {
			write("grpc_connected", boolNumber(c.GetConnected()))
			write("grpc_reconnects_total", c.GetReconnects())
			write("grpc_errors_total", c.GetErrors())
		}
		if c := m.GetBroker(); c != nil {
			write("broker_connected", boolNumber(c.GetConnected()))
			write("broker_reconnects_total", c.GetReconnects())
			write("broker_errors_total", c.GetErrors())
		}
		if r := m.GetRuntime(); r != nil {
			write("uptime_seconds", r.GetUptimeSeconds())
			write("goroutines", r.GetGoroutines())
			write("memory_bytes", r.GetMemoryBytes())
			write("websocket_clients", r.GetWebsocketClients())
			write("backpressure_level", r.GetBackpressureLevel())
			write("websocket_dropped_total", r.GetWebsocketDropped())
		}
	})
}

func boolNumber(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
