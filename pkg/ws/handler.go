package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Only for testing
	},
}

func (h *Hub) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade failed:", err)
		return
	}

	client := NewClient(h, conn)

	// Register client in hub goroutine
	h.register <- client

	// Start write pump (single writer goroutine)
	go client.writePump()

	// Start read pump to detect disconnects
	go client.readPump()
}
