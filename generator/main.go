package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Event struct {
	Sensor    string  `json:"sensor"`
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	opts := mqtt.NewClientOptions().
		AddBroker("tcp://mosquitto:1883").
		SetClientID("data-generator")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Println("Generator connected to MQTT")

	for {
		event := Event{
			Sensor:    "sensor-" + string(rune('A'+rand.Intn(5))),
			Value:     10 + rand.Float64()*25,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		b, _ := json.Marshal(event)
		client.Publish("sensors/temperature", 0, false, b)

		log.Println("Published:", string(b))
		time.Sleep(1 * time.Second)
	}
}
