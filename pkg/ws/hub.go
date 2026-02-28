package ws

import (
	"encoding/json"
)

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

type Broadcaster interface {
	BroadcastEvent(any)
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 1024),
	}
}

func (h *Hub) BroadcastEvent(e any) {
	data, _ := json.Marshal(e)
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// slow client, disconnect
					close(c.send)
					delete(h.clients, c)
				}
			}
		}
	}
}
