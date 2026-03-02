package generator

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
)

var interval time.Duration

type Publisher struct {
	B broker.Broker
}

func (p *Publisher) SendEvent(e domain.Sensor) error {
	data, _ := json.Marshal(e)
	return p.B.Publish("sensors/temperature", data)
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {

		select {
		case <-ctx.Done():
			log.Println("Publisher stopping: context cancelled")
			return

		case <-ticker.C:
			event := domain.Sensor{
				Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
				Value:     10 + rand.Float64()*20,
				Timestamp: time.Now().UTC(),
			}

			if err := p.SendEvent(event); err != nil {
				log.Printf("Failed to send event: %v", err)
			}
		}
	}
}

func init() {
	rateStr := os.Getenv("PUBLISH_RATE")
	if rateStr == "" {
		rateStr = "1"
	}
	rate := 1 // events/sec
	if v, err := strconv.Atoi(rateStr); err == nil && v > 0 {
		rate = v
	}

	interval = time.Second / time.Duration(rate)
	log.Printf("Publishing at %d events/sec (interval %v)", rate, interval)
}

func NewPublisher() (*Publisher, error) {
	clientId := os.Getenv("GENERATOR_ID")
	if clientId == "" {
		return nil, errors.New("GENERATOR_ID not defined")
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		return nil, err
	}

	return &Publisher{B: mclient}, nil
}
