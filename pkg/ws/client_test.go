package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWritePumpRelaysMessagesToSocket(t *testing.T) {
	upgrader := websocket.Upgrader{}
	serverConnCh := make(chan *websocket.Conn, 1)
	messageCh := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
		_, payload, err := conn.ReadMessage()
		if err == nil {
			messageCh <- string(payload)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	serverConn := <-serverConnCh
	defer serverConn.Close()

	client := NewClient(NewHub(), conn)
	go client.writePump()

	client.send <- []byte("hello")

	select {
	case msg := <-messageCh:
		require.Equal(t, "hello", msg)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for websocket message")
	}

	close(client.send)
}
