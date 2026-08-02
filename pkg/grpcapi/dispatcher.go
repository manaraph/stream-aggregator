package grpcapi

import (
	"encoding/json"

	"github.com/manaraph/stream-aggregator/pkg/events"
	"github.com/manaraph/stream-aggregator/pkg/ws"
)

// Dispatcher delivers events received by the gRPC services.
type Dispatcher interface {
	Publish(events.Message)
}

// WebSocketDispatcher serializes events and sends them to WebSocket clients.
type WebSocketDispatcher struct {
	hub *ws.Hub
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
