package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

const pongWait = 60 * time.Second

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewClient(h *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// readPump only detects disconnects
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
