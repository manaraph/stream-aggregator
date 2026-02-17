package gateway

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
}

var upgrader = websocket.Upgrader{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{conn: c}
	clients[client] = true
}

func StartBroadcastLoop() {
	for e := range broadcast {
		data, _ := json.Marshal(e)
		for c := range clients {
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("WS error:", err)
				c.conn.Close()
				delete(clients, c)
				log.Println("WS disconnected")
			}
		}
	}
}
