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

	h.BroadcastEvent(map[string]string{"msg": "hello"})

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
