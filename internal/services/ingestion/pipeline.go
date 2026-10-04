package ingestion

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/domain"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/protobuf/proto"
)

const (
	defaultBatchSize     = 1000
	defaultBatchInterval = 3 * time.Second
	defaultQueueSize     = 10000
)

func (p *Processor) initPipeline() {
	batchSize := positiveIntEnv("DB_BATCH_SIZE", defaultBatchSize)
	queueSize := positiveIntEnv("INGESTION_QUEUE_SIZE", defaultQueueSize)
	batchInterval := durationEnv("DB_BATCH_INTERVAL", defaultBatchInterval)

	p.eventQueue = make(chan queuedReading, queueSize)
	if p.ctx == nil {
		p.ctx = context.Background()
	}
	if p.writerCtx == nil {
		p.writerCtx, p.writerStop = context.WithCancel(context.Background())
	}

	log.Printf("Starting ingestion pipeline: queue=%d batch_size=%d batch_interval=%s", queueSize, batchSize, batchInterval)
	go p.batchWriter(batchSize, batchInterval)
	go p.queueStatus()
}

func positiveIntEnv(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (p *Processor) enqueueEvent(e domain.Sensor, message mqtt.Message) {
	p.wg.Add(1)
	select {
	case p.eventQueue <- queuedReading{reading: e, message: message}:
		p.recordQueueHighWaterMark(uint32(len(p.eventQueue)))
	case <-p.ctx.Done():
		p.wg.Done()
	}
}

func (p *Processor) batchWriter(maxBatch int, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	batch := make([]queuedReading, 0, maxBatch)

	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		readings := make([]domain.Sensor, len(batch))
		for i := range batch {
			readings[i] = batch[i].reading
		}

		backoff := 100 * time.Millisecond
		for {
			ctx, cancel := context.WithTimeout(p.writerCtx, 10*time.Second)
			err := p.store.InsertBatch(ctx, readings)
			cancel()
			if err == nil {
				for _, item := range batch {
					p.ForwardEvent(item.reading)
					item.message.Ack()
					atomic.AddUint64(&p.processed, 1)
					p.wg.Done()
				}
				batch = batch[:0]
				return true
			}
			log.Printf("PostgreSQL batch write failed; retrying %d readings: %v", len(batch), err)
			select {
			case <-p.writerCtx.Done():
				for range batch {
					p.wg.Done()
				}
				batch = batch[:0]
				return false
			case <-time.After(backoff):
			}
			if backoff < 5*time.Second {
				backoff *= 2
			}
		}
	}

	for {
		select {
		case item := <-p.eventQueue:
			batch = append(batch, item)
			if len(batch) >= maxBatch && !flush() {
				return
			}
		case <-ticker.C:
			if !flush() {
				return
			}
		case <-p.ctx.Done():
			for {
				select {
				case item := <-p.eventQueue:
					batch = append(batch, item)
					if len(batch) == maxBatch && !flush() {
						return
					}
				default:
					flush()
					return
				}
			}
		}
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
			Processed:   proto.Uint64(processed),
			Dropped:     proto.Uint64(dropped),
			Used:        proto.Uint32(uint32(used)),
			Capacity:    proto.Uint32(uint32(capacity)),
			MaxUsed:     proto.Uint32(atomic.LoadUint32(&p.maxUsed)),
			Utilization: proto.Float64(percent),
		},
		Throughput: &streamv1.ThroughputMetrics{IngestionRate: proto.Float64(rate)},
		Grpc:       &streamv1.ConnectionMetrics{Connected: proto.Bool(p.S != nil && p.M != nil), Errors: proto.Uint32(atomic.LoadUint32(&p.grpcErrors))},
		Broker:     &streamv1.ConnectionMetrics{Connected: proto.Bool(p.B != nil)},
	})

	return processed
}
