package grpcapi

import (
	"encoding/json"
	"sync"

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
	hub     *ws.Hub
	mu      sync.Mutex
	metrics *streamv1.IngestMetricsRequest
}

func NewWebSocketDispatcher(h *ws.Hub) *WebSocketDispatcher {
	return &WebSocketDispatcher{
		hub: h,
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
	if d.metrics == nil {
		d.metrics = &streamv1.IngestMetricsRequest{}
	}
	proto.Merge(d.metrics, update)
	snapshot := proto.Clone(d.metrics).(*streamv1.IngestMetricsRequest)
	d.mu.Unlock()

	d.Publish(events.NewMetricsMessage(snapshot))
}
