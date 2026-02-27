package ingestion

import (
	"log"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/manaraph/stream-aggregator/internal/domain"
)

var (
	eventQueue  chan domain.Sensor
	workerCount = 4
	queueSize   = 1000
	processed   uint64
	dropped     uint64
)

func init() {
	if v := os.Getenv("INGESTION_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			workerCount = n
		}
	}
	if v := os.Getenv("INGESTION_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			queueSize = n
		}
	}

	eventQueue = make(chan domain.Sensor, queueSize)
}

func StartPipeline() {
	log.Printf("Starting ingestion pipeline: workers=%d queue=%d", workerCount, queueSize)

	for i := 0; i < workerCount; i++ {
		go worker(i)
	}
}

func worker(id int) {
	log.Printf("Worker %d started", id)
	for e := range eventQueue {
		forwardEvent(e)
	}
}

func EnqueueEvent(e domain.Sensor) {
	select {
	case eventQueue <- e:
		atomic.AddUint64(&processed, 1)
	default:
		// queue full -> drop event
		log.Println("WARNING: ingestion queue full, dropping event")
		atomic.AddUint64(&dropped, 1)
	}
}

// func () {
//     ticker := time.NewTicker(2 * time.Second)
//     for range ticker.C {
//         used := len(sensorQueue)
//         capacity := cap(sensorQueue)
//         percent := float64(used) / float64(capacity) * 100
//         p := atomic.LoadUint64(&processed)
//         d := atomic.LoadUint64(&dropped)

//         log.Printf("QUEUE %d/%d (%.1f%%) processed=%d dropped=%d",
//             used, capacity, percent, p, d)
//     }
// }()
