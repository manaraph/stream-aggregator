package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestHubDropsMessagesWhenQueueIsFull(t *testing.T) {
	h := NewHub()
	for i := 0; i < 1025; i++ {
		h.Broadcast([]byte("x"))
	}

	assert.Equal(t, uint64(1), h.dropped)
}

func TestHubRemovesSlowClientOnBroadcastFailure(t *testing.T) {
	h := NewHub()
	go h.Run()

	client := &Client{hub: h, send: make(chan []byte, 1)}
	client.send <- []byte("one")
	h.register <- client

	assert.Eventually(t, func() bool {
		return h.Stats().Clients == 1
	}, time.Second, 10*time.Millisecond)

	h.events <- []byte("msg")
	assert.Eventually(t, func() bool {
		return h.Stats().Clients == 0
	}, time.Second, 10*time.Millisecond)
}
