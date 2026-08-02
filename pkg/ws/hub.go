package ws

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	events     chan []byte
}

type Broadcaster interface {
	Broadcast([]byte)
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

		case msg := <-h.events:
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
