package generator

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/client"
	"github.com/manaraph/stream-aggregator/internal/domain"
)

var interval time.Duration

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

func Start() {
	clientId := os.Getenv("GENERATOR_ID")
	if clientId == "" {
		log.Fatalf("GENERATOR_ID not defined")
		return
	}

	mclient, err := client.InitMQTTClient(clientId)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
		return
	}

	publishSensorData(mclient)
}

func publishSensorData(client mqtt.Client) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		event := domain.Sensor{
			Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
			Value:     10 + rand.Float64()*20,
			Timestamp: time.Now().UTC(),
		}

		b, _ := json.Marshal(event)
		client.Publish("sensors/temperature", 0, false, b)

		log.Println("Published:", string(b))
	}
}
