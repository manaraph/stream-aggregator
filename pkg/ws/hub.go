package ws

import "sync/atomic"

type Hub struct {
	clients     map[*Client]struct{}
	register    chan *Client
	unregister  chan *Client
	events      chan []byte
	clientCount uint32
	dropped     uint64
	delivered   uint64
}

type Broadcaster interface {
	Broadcast([]byte)
}

type Stats struct {
	Clients           uint32
	BackpressureLevel uint32
	Delivered         uint64
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		events:     make(chan []byte, 1024),
	}
}

func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.events <- msg:
	default:
		atomic.AddUint64(&h.dropped, 1)
	}
}

func (h *Hub) Stats() Stats {
	return Stats{
		Clients:           atomic.LoadUint32(&h.clientCount),
		BackpressureLevel: uint32(len(h.events)),
		Delivered:         atomic.LoadUint64(&h.delivered),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = struct{}{}
			atomic.StoreUint32(&h.clientCount, uint32(len(h.clients)))

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				atomic.StoreUint32(&h.clientCount, uint32(len(h.clients)))
			}

		case msg := <-h.events:
			for c := range h.clients {
				select {
				case c.send <- msg:
					atomic.AddUint64(&h.delivered, 1)
				default:
					// slow client, disconnect
					close(c.send)
					delete(h.clients, c)
					atomic.StoreUint32(&h.clientCount, uint32(len(h.clients)))
				}
			}
		}
	}
}
