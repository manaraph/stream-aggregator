package events

import (
	"encoding/json"
	"strings"
	"testing"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsMessageUsesCamelCaseJSON(t *testing.T) {
	message := NewMetricsMessage(&streamv1.IngestMetricsRequest{
		Queue:      &streamv1.QueueMetrics{MaxUsed: 3},
		Throughput: &streamv1.ThroughputMetrics{WebsocketRate: 2.5},
		Runtime:    &streamv1.RuntimeMetrics{UptimeSeconds: 10, BackpressureLevel: 4},
	})

	payload, err := json.Marshal(message)
	assert.NoError(t, err)
	json := string(payload)
	assert.True(t, strings.Contains(json, `"maxUsed":3`))
	assert.True(t, strings.Contains(json, `"websocketRate":2.5`))
	assert.True(t, strings.Contains(json, `"uptimeSeconds":10`))
	assert.True(t, strings.Contains(json, `"backpressureLevel":4`))
	assert.False(t, strings.Contains(json, "max_used"))
	assert.False(t, strings.Contains(json, "websocket_rate"))
}
