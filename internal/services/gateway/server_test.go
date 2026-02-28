package gateway

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayStarts(t *testing.T) {
	g, err := NewGateway("127.0.0.1:0", "127.0.0.1:0")
	require.NoError(t, err)
	require.NotNil(t, g.Hub)

	time.Sleep(50 * time.Millisecond) // ensure goroutines started
}
