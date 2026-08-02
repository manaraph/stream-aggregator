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

	srv := httptest.NewServer(http.HandlerFunc(h.handler))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:]

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	h.Broadcast([]byte(`{"hello": "world"}`))

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if len(msg) == 0 {
		t.Fatal("expected message")
	}
}

func TestRegisterRoutesRegistersSingleWebSocketEndpoint(t *testing.T) {
	h := NewHub()
	mux := http.NewServeMux()
	h.RegisterRoute(mux)

	for _, path := range []string{"/ws/sensors", "/ws/metrics"} {
		_, pattern := mux.Handler(httptest.NewRequest(http.MethodGet, path, nil))
		if pattern != "" {
			t.Fatalf("expected %s not to be registered, got %q", path, pattern)
		}
	}

	_, pattern := mux.Handler(httptest.NewRequest(http.MethodGet, "/ws", nil))
	if pattern != "/ws" {
		t.Fatalf("expected /ws to be registered, got %q", pattern)
	}
}
