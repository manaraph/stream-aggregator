package generator

import (
	"log"
	"os"

	"github.com/manaraph/stream-aggregator/pkg/broker"
)

func Start() {
	clientId := os.Getenv("GENERATOR_ID")
	if clientId == "" {
		log.Fatalf("GENERATOR_ID not defined")
		return
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
		return
	}

	p := &Publisher{B: mclient}
	p.Run()
}
