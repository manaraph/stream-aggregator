package gateway

import (
	"testing"
	"time"

	"github.com/manaraph/stream-aggregator/pkg/events"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGatewayStarts(t *testing.T) {
	g, err := NewGateway("127.0.0.1:0", "127.0.0.1:0")
	require.NoError(t, err)
	require.NotNil(t, g.Hub)

	time.Sleep(50 * time.Millisecond) // ensure goroutines started
}

func TestAllocatedMemoryReturnsValue(t *testing.T) {
	mem := allocatedMemory()
	require.Greater(t, mem, uint64(0))
}

type stubDispatcher struct {
	metrics []*streamv1.IngestMetricsRequest
}

func (s *stubDispatcher) Publish(events.Message) {}

func (s *stubDispatcher) PublishMetrics(metrics *streamv1.IngestMetricsRequest) {
	if metrics != nil {
		s.metrics = append(s.metrics, proto.Clone(metrics).(*streamv1.IngestMetricsRequest))
	}
}

func TestPublishMetricsWithTickerPublishesRuntimeSnapshot(t *testing.T) {
	hub := ws.NewHub()
	dispatcher := &stubDispatcher{}
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	close(tick)

	publishMetricsWithTicker(hub, dispatcher, tick, time.Now())

	require.Len(t, dispatcher.metrics, 1)
	metrics := dispatcher.metrics[0]
	assert.NotNil(t, metrics.GetThroughput())
	assert.NotNil(t, metrics.GetRuntime())
	assert.Equal(t, uint32(0), metrics.GetRuntime().GetWebsocketClients())
}
