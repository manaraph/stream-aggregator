package generator

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/stretchr/testify/assert"
)

func TestPublisher_Run(t *testing.T) {
	interval = 10 * time.Millisecond
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go pub.Run(ctx)

	for i := 0; i < 3; i++ {
		select {
		case msg := <-fake.Messages:
			var s domain.Sensor
			err := json.Unmarshal(msg, &s)
			assert.NoError(t, err)
			assert.Contains(t, s.Sensor, "sensor-")
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Failed to receive event %d", i)
		}
	}
}

func TestPublisher_Cancellation(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	pub.Run(ctx)
	duration := time.Since(start)

	assert.True(t, duration >= 50*time.Millisecond)
	assert.True(t, duration < 100*time.Millisecond, "Run took too long to exit")
}

func TestSendEvent(t *testing.T) {
	fake := broker.NewFakeBroker()
	pub := &Publisher{B: fake}

	event := domain.Sensor{
		Sensor:    "sensor-A",
		Value:     42.0,
		Timestamp: time.Now().UTC(),
	}

	err := pub.SendEvent(event)
	assert.NoError(t, err)

	select {
	case msg := <-fake.Messages:
		var got domain.Sensor
		err := json.Unmarshal(msg, &got)
		assert.NoError(t, err)
		assert.Equal(t, "sensor-A", got.Sensor)
		assert.Equal(t, 42.0, got.Value)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout: Broker never received the message")
	}
}

func TestNewPublisherFromEnv(t *testing.T) {
	t.Run("Fails when env vars are missing", func(t *testing.T) {
		os.Unsetenv("GENERATOR_ID")
		os.Unsetenv("MQTT_BROKER")

		pub, err := NewPublisherFromEnv()
		assert.Nil(t, pub)
		assert.Error(t, err)
	})

	t.Run("Fails with invalid broker URL", func(t *testing.T) {
		os.Setenv("GENERATOR_ID", "test-gen")
		os.Setenv("MQTT_BROKER", "tcp://invalid-address:9999")
		defer os.Clearenv()

		pub, err := NewPublisherFromEnv()
		assert.Nil(t, pub)
		assert.Error(t, err)
	})
}
