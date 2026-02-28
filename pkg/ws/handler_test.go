package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHandlerBroadcast(t *testing.T) {
	h := NewHub()
	go h.Run()

	srv := httptest.NewServer(http.HandlerFunc(h.Handler))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:]

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	h.BroadcastEvent(map[string]string{"hello": "world"})

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if len(msg) == 0 {
		t.Fatal("expected message")
	}
}
