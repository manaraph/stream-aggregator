package ws

import (
	"sync"
	"testing"
)

func TestHubRaceStress(t *testing.T) {
	h := NewHub()
	go h.Run()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			c := &Client{
				hub:  h,
				send: make(chan []byte, 256),
			}

			h.register <- c

			for j := 0; j < 100; j++ {
				h.BroadcastEvent(map[string]int{"n": j})
			}

			h.unregister <- c
		}()
	}

	wg.Wait()
}

func BenchmarkHubBroadcast(b *testing.B) {
	h := NewHub()
	go h.Run()

	c := &Client{hub: h, send: make(chan []byte, 1024)}
	h.register <- c

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.BroadcastEvent(map[string]int{"n": i})
	}
}
