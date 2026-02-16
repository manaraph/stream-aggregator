package generator

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/client"
	"github.com/manaraph/stream-aggregator/internal/domain"
)

func Start() {
	mclient, err := client.InitMQTTClient("data-generator")
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
		return
	}
	initSensorDataGenerator(*mclient)
}

func initSensorDataGenerator(client mqtt.Client) {
	for {
		e := domain.Sensor{
			Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
			Value:     10 + rand.Float64()*20,
			Timestamp: time.Now().UTC(),
		}

		b, _ := json.Marshal(e)
		client.Publish("sensors/temperature", 0, false, b)

		log.Println("Published:", string(b))
		time.Sleep(time.Second)
	}
}
