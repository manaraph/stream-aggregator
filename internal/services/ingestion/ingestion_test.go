package ingestion

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProcessor_Config(t *testing.T) {
	// Clean up env before each sub-test
	os.Unsetenv("INGESTION_ID")
	os.Unsetenv("GATEWAY_ADDR")
	os.Unsetenv("MQTT_BROKER")

	t.Run("Missing INGESTION_ID", func(t *testing.T) {
		p, err := NewProcessor()
		assert.Nil(t, p)
		assert.EqualError(t, err, "INGESTION_ID not defined")
	})

	t.Run("Missing GATEWAY_ADDR", func(t *testing.T) {
		os.Setenv("INGESTION_ID", "test-id")
		defer os.Unsetenv("INGESTION_ID")

		p, err := NewProcessor()
		assert.Nil(t, p)
		assert.EqualError(t, err, "GATEWAY_ADDR not defined")
	})

	t.Run("MQTT Connection Failure", func(t *testing.T) {
		os.Setenv("INGESTION_ID", "test-id")
		os.Setenv("GATEWAY_ADDR", "localhost:50051")
		// Point MQTT to a port that is definitely closed
		os.Setenv("MQTT_BROKER", "tcp://localhost:1234")

		defer os.Clearenv()

		p, err := NewProcessor()
		assert.Nil(t, p)
		assert.Error(t, err, "Should fail because MQTT broker is unreachable")
	})
}
