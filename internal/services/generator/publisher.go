package generator

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/client"
	"github.com/manaraph/stream-aggregator/internal/domain"
)

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
	for {
		e := domain.Sensor{
			Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
			Value:     10 + rand.Float64()*20,
			Timestamp: time.Now().UTC(),
		}

		b, _ := json.Marshal(e)
		client.Publish("sensors/temperature", 0, false, b)

		log.Println("Published:", string(b))
		time.Sleep(3 * time.Second)
	}
}
