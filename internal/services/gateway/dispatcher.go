package gateway

import (
	"encoding/json"

	"github.com/manaraph/stream-aggregator/pkg/events"
	"github.com/manaraph/stream-aggregator/pkg/ws"
)

type Dispatcher struct {
	hub *ws.Hub
}

func NewDispatcher(h *ws.Hub) *Dispatcher {
	return &Dispatcher{
		hub: h,
	}
}

func (d *Dispatcher) Publish(message events.Message) {
	b, err := json.Marshal(message)
	if err != nil {
		return
	}

	d.hub.Broadcast(b)
}
