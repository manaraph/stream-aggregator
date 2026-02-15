package main

import (
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883"
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("ingestion-service")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Println("Connected to MQTT broker:", broker)

	client.Subscribe("sensors/#", 0, func(c mqtt.Client, m mqtt.Message) {
		log.Printf("Ingested topic=%s payload=%s", m.Topic(), string(m.Payload()))
	})

	select {}
}
