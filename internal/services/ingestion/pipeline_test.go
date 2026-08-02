package ingestion

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcessor_Flow(t *testing.T) {
	mockStream := new(MockStream)

	p := &Processor{
		B:  broker.NewFakeBroker(),
		S:  mockStream,
		WG: sync.WaitGroup{},
	}

	mockStream.On("Send", mock.Anything).Return(nil)

	p.initPipeline()

	event := domain.Sensor{Sensor: "test-sensor", Value: 10.5}
	p.enqueueEvent(event)

	p.WG.Wait()

	assert.Equal(t, uint64(1), atomic.LoadUint64(&p.processed))
	mockStream.AssertExpectations(t)
}

func TestProcessor_QueueFull(t *testing.T) {
	p := &Processor{
		eventQueue: make(chan domain.Sensor, 1),
	}

	e := domain.Sensor{Sensor: "s1"}

	p.enqueueEvent(e)
	assert.Equal(t, uint64(1), atomic.LoadUint64(&p.processed))

	p.enqueueEvent(e)
	assert.Equal(t, uint64(1), atomic.LoadUint64(&p.dropped))
}

func TestProcessor_TracksQueueHighWaterMark(t *testing.T) {
	p := &Processor{eventQueue: make(chan domain.Sensor, 2)}

	p.enqueueEvent(domain.Sensor{Sensor: "s1"})
	p.enqueueEvent(domain.Sensor{Sensor: "s2"})

	assert.Equal(t, uint32(2), atomic.LoadUint32(&p.maxUsed))
}

func TestProcessor_InitPipelineConfig(t *testing.T) {
	os.Setenv("INGESTION_QUEUE_SIZE", "555")
	defer os.Unsetenv("INGESTION_QUEUE_SIZE")

	p := &Processor{}
	p.initPipeline()

	assert.Equal(t, 555, cap(p.eventQueue))
}
