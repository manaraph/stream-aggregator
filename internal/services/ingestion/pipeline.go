package ingestion

import (
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
)

func (p *Processor) initPipeline() {
	workerCount := 4
	queueSize := 1000

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

	p.eventQueue = make(chan domain.Sensor, queueSize)

	log.Printf("Starting ingestion pipeline: workers=%d queue=%d", workerCount, queueSize)

	for i := 0; i < workerCount; i++ {
		go p.worker(i)
	}

	go p.queueStatus()
}

func (p *Processor) worker(id int) {
	log.Printf("Worker %d started", id)

	for e := range p.eventQueue {
		p.ForwardEvent(e)
	}
}

func (p *Processor) EnqueueEvent(e domain.Sensor) {
	select {
	case p.eventQueue <- e:
		atomic.AddUint64(&p.processed, 1)
	default:
		log.Println("WARNING: ingestion queue full, dropping event")
		atomic.AddUint64(&p.dropped, 1)
	}
}

func (p *Processor) queueStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		used := len(p.eventQueue)
		capacity := cap(p.eventQueue)
		percent := float64(used) / float64(capacity) * 100

		log.Printf("QUEUE %d/%d (%.1f%%) processed=%d dropped=%d",
			used,
			capacity,
			percent,
			atomic.LoadUint64(&p.processed),
			atomic.LoadUint64(&p.dropped),
		)
	}
}
