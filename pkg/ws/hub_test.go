package ws

import (
	"testing"
	"time"
)

func TestHubRegisterBroadcastUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()

	// mock client
	c := &Client{
		hub:  h,
		send: make(chan []byte, 10),
	}

	h.register <- c
	time.Sleep(10 * time.Millisecond)

	h.Broadcast([]byte(`{"msg": "hello"}`))

	select {
	case msg := <-c.send:
		if string(msg) == "" {
			t.Fatal("expected message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}

	h.unregister <- c
	time.Sleep(10 * time.Millisecond)

	_, ok := <-c.send
	if ok {
		t.Fatal("expected channel closed")
	}
}

func TestHubStatsReportsRuntimeState(t *testing.T) {
	h := NewHub()
	go h.Run()

	stats := h.Stats()
	if stats.Clients != 0 || stats.Delivered != 0 || stats.BackpressureLevel != 0 {
		t.Fatalf("expected empty stats, got %+v", stats)
	}

	h.Broadcast([]byte(`{"msg": "hello"}`))
	stats = h.Stats()
	if stats.BackpressureLevel == 0 {
		t.Fatal("expected backpressure level to reflect queued events")
	}
}
