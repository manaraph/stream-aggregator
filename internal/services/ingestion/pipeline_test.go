package ingestion

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testBatchStore struct {
	mu       sync.Mutex
	calls    int
	failures int
	called   chan struct{}
}

func (s *testBatchStore) InsertBatch(context.Context, []domain.Sensor) error {
	s.mu.Lock()
	s.calls++
	fail := s.failures > 0
	if fail {
		s.failures--
	}
	called := s.called
	s.mu.Unlock()
	if called != nil {
		select {
		case called <- struct{}{}:
		default:
		}
	}
	if fail {
		return errors.New("temporary database failure")
	}
	return nil
}

func (s *testBatchStore) Close() {}

func (s *testBatchStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestProcessor_Flow(t *testing.T) {
	mockStream := new(MockStream)
	p := &Processor{
		B:     broker.NewFakeBroker(),
		store: &testBatchStore{},
		S:     mockStream,
		wg:    sync.WaitGroup{},
	}
	mockStream.On("Send", mock.Anything).Return(nil)
	t.Setenv("DB_BATCH_SIZE", "1")
	t.Setenv("DB_BATCH_INTERVAL", "10ms")
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.initPipeline()

	message := &broker.MockMessage{}
	p.enqueueEvent(domain.Sensor{Sensor: "test-sensor", Value: 10.5}, message)
	p.wg.Wait()

	assert.Equal(t, uint64(1), atomic.LoadUint64(&p.processed))
	assert.True(t, message.Acked)
	mockStream.AssertExpectations(t)
	assert.NoError(t, p.Close(context.Background()))
}

func TestProcessor_QueueBackpressureDoesNotDrop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{eventQueue: make(chan queuedReading, 1), ctx: ctx}
	event := domain.Sensor{Sensor: "s1"}

	p.enqueueEvent(event, &broker.MockMessage{})
	cancel()
	p.enqueueEvent(event, &broker.MockMessage{})

	assert.Equal(t, 1, len(p.eventQueue))
	assert.Equal(t, uint64(0), atomic.LoadUint64(&p.dropped))
	p.wg.Done()
	p.wg.Wait()
}

func TestProcessor_TracksQueueHighWaterMark(t *testing.T) {
	p := &Processor{eventQueue: make(chan queuedReading, 2), ctx: context.Background()}
	p.enqueueEvent(domain.Sensor{Sensor: "s1"}, &broker.MockMessage{})
	p.enqueueEvent(domain.Sensor{Sensor: "s2"}, &broker.MockMessage{})

	assert.Equal(t, uint32(2), atomic.LoadUint32(&p.maxUsed))
	p.wg.Done()
	p.wg.Done()
}

func TestBatchWriterFlushesPartialBatchOnInterval(t *testing.T) {
	store := &testBatchStore{}
	stream := new(MockStream)
	stream.On("Send", mock.Anything).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{store: store, S: stream, ctx: ctx, cancel: cancel}
	t.Setenv("DB_BATCH_SIZE", "10")
	t.Setenv("DB_BATCH_INTERVAL", "10ms")
	p.initPipeline()
	message := &broker.MockMessage{}
	p.enqueueEvent(domain.Sensor{Sensor: "s1"}, message)

	completed := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(completed)
	}()
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("timed batch flush did not complete")
	}

	assert.Equal(t, 1, store.callCount())
	assert.True(t, message.Acked)
	assert.NoError(t, p.Close(context.Background()))
}

func TestBatchWriterRetriesFailedBatch(t *testing.T) {
	store := &testBatchStore{failures: 1}
	stream := new(MockStream)
	stream.On("Send", mock.Anything).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{store: store, S: stream, ctx: ctx, cancel: cancel}
	t.Setenv("DB_BATCH_SIZE", "1")
	t.Setenv("DB_BATCH_INTERVAL", "1h")
	p.initPipeline()
	message := &broker.MockMessage{}
	p.enqueueEvent(domain.Sensor{Sensor: "s1"}, message)

	completed := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(completed)
	}()
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("batch retry did not complete")
	}

	assert.Equal(t, 2, store.callCount())
	assert.True(t, message.Acked)
	assert.NoError(t, p.Close(context.Background()))
}

func TestBatchWriterLeavesMessageUnackedWhenStoppedDuringRetry(t *testing.T) {
	store := &testBatchStore{failures: 10, called: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	writerCtx, writerStop := context.WithCancel(context.Background())
	p := &Processor{
		store: store, ctx: ctx, cancel: cancel,
		writerCtx: writerCtx, writerStop: writerStop,
		eventQueue: make(chan queuedReading, 1),
	}
	go p.batchWriter(1, time.Hour)
	message := &broker.MockMessage{}
	p.enqueueEvent(domain.Sensor{Sensor: "s1"}, message)
	<-store.called
	writerStop()
	completed := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(completed)
	}()
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("batch writer did not stop after cancellation")
	}
	assert.False(t, message.Acked)
	cancel()
}

func TestProcessor_InitPipelineConfig(t *testing.T) {
	t.Setenv("INGESTION_QUEUE_SIZE", "555")
	ctx, cancel := context.WithCancel(context.Background())
	p := &Processor{ctx: ctx, cancel: cancel}
	p.initPipeline()

	assert.Equal(t, 555, cap(p.eventQueue))
	p.cancel()
	p.writerStop()
}

func TestProcessor_ReportQueueStatusIncludesMetrics(t *testing.T) {
	metricsStream := new(MockMetricsStream)
	p := &Processor{eventQueue: make(chan queuedReading, 2), M: metricsStream}
	metricsStream.On("Send", mock.Anything).Return(nil).Once()
	p.processed, p.dropped, p.maxUsed, p.grpcErrors = 3, 1, 2, 4
	p.B = broker.NewFakeBroker()
	p.S = new(MockStream)

	assert.Equal(t, uint64(3), p.reportQueueStatus(1, 5*time.Second))
	metricsStream.AssertExpectations(t)
}

func TestProcessor_ReportQueueStatusHandlesEmptyCapacity(t *testing.T) {
	p := &Processor{eventQueue: make(chan queuedReading)}
	p.processed, p.dropped, p.maxUsed = 5, 2, 1

	assert.Equal(t, uint64(5), p.reportQueueStatus(0, 5*time.Second))
}
