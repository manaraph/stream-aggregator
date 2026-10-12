package grpcapi

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/manaraph/stream-aggregator/pkg/events"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"google.golang.org/protobuf/proto"
)

// Dispatcher delivers events received by the gRPC services.
type Dispatcher interface {
	Publish(events.Message)
	PublishMetrics(*streamv1.IngestMetricsRequest)
}

// WebSocketDispatcher serializes events and sends them to WebSocket clients.
type WebSocketDispatcher struct {
	hub         *ws.Hub
	mu          sync.Mutex
	metrics     *streamv1.IngestMetricsRequest
	lastReports map[string]time.Time
}

func NewWebSocketDispatcher(h *ws.Hub) *WebSocketDispatcher {
	return &WebSocketDispatcher{
		hub:         h,
		lastReports: make(map[string]time.Time),
	}
}

func (d *WebSocketDispatcher) Publish(message events.Message) {
	b, err := json.Marshal(message)
	if err != nil {
		return
	}

	d.hub.Broadcast(b)
}

func (d *WebSocketDispatcher) PublishMetrics(update *streamv1.IngestMetricsRequest) {
	if update == nil {
		return
	}

	d.mu.Lock()
	now := time.Now()
	if update.GetThroughput() != nil && update.GetThroughput().PublishRate != nil {
		d.lastReports["generator"] = now
	}
	if update.GetQueue() != nil {
		d.lastReports["ingestion"] = now
	}
	if update.GetRuntime() != nil {
		d.lastReports["gateway"] = now
	}
	if d.metrics == nil {
		d.metrics = &streamv1.IngestMetricsRequest{}
	}
	proto.Merge(d.metrics, update)
	snapshot := proto.Clone(d.metrics).(*streamv1.IngestMetricsRequest)
	d.mu.Unlock()

	d.Publish(events.NewMetricsMessage(snapshot))
}

// LastReports returns the latest metric receive time for each service.
func (d *WebSocketDispatcher) LastReports() map[string]time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	reports := make(map[string]time.Time, len(d.lastReports))
	for service, timestamp := range d.lastReports {
		reports[service] = timestamp
	}
	return reports
}

// MetricsSnapshot returns a copy of the most recently merged metrics.
func (d *WebSocketDispatcher) MetricsSnapshot() *streamv1.IngestMetricsRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.metrics == nil {
		return nil
	}
	return proto.Clone(d.metrics).(*streamv1.IngestMetricsRequest)
}
