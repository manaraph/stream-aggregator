package ingestion

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
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
	if p.ctx == nil {
		p.ctx = context.Background()
	}

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
		p.WG.Done()
	}
}

func (p *Processor) enqueueEvent(e domain.Sensor) {
	p.WG.Add(1)

	select {
	case p.eventQueue <- e:
		atomic.AddUint64(&p.processed, 1)
		p.recordQueueHighWaterMark(uint32(len(p.eventQueue)))
	default:
		log.Println("WARNING: ingestion queue full, dropping event")
		atomic.AddUint64(&p.dropped, 1)
		p.WG.Done()
	}
}

func (p *Processor) recordQueueHighWaterMark(used uint32) {
	for {
		maxUsed := atomic.LoadUint32(&p.maxUsed)
		if used <= maxUsed || atomic.CompareAndSwapUint32(&p.maxUsed, maxUsed, used) {
			return
		}
	}
}

func (p *Processor) queueStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var previousProcessed uint64
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			previousProcessed = p.reportQueueStatus(previousProcessed, 5*time.Second)
		}
	}
}

func (p *Processor) reportQueueStatus(previousProcessed uint64, interval time.Duration) uint64 {
	used := len(p.eventQueue)
	capacity := cap(p.eventQueue)
	percent := 0.0
	if capacity > 0 {
		percent = float64(used) / float64(capacity) * 100
	}

	processed := atomic.LoadUint64(&p.processed)
	dropped := atomic.LoadUint64(&p.dropped)
	rate := float64(processed-previousProcessed) / interval.Seconds()

	p.ForwardMetrics(&streamv1.IngestMetricsRequest{
		Queue: &streamv1.QueueMetrics{
			Processed:   processed,
			Dropped:     dropped,
			Used:        uint32(used),
			Capacity:    uint32(capacity),
			MaxUsed:     atomic.LoadUint32(&p.maxUsed),
			Utilization: percent,
		},
		Throughput: &streamv1.ThroughputMetrics{IngestionRate: rate},
		Grpc:       &streamv1.ConnectionMetrics{Connected: p.S != nil && p.M != nil, Errors: atomic.LoadUint32(&p.grpcErrors)},
		Broker:     &streamv1.ConnectionMetrics{Connected: p.B != nil},
	})

	return processed
}
