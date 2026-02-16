package ingestion

import (
	"encoding/json"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/client"
	"github.com/manaraph/stream-aggregator/internal/domain"
)

func Start() {
	clientId := os.Getenv("INGESTION_ID")
	if clientId == "" {
		log.Fatalf("INGESTION_ID not defined")
		return
	}

	mclient, err := client.InitMQTTClient(clientId)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
		return
	}

	subscribeToSensors(mclient)

	select {}
}

func subscribeToSensors(mclient mqtt.Client) {
	mclient.Subscribe("sensors/#", 0, handleMessage)
}

func handleMessage(c mqtt.Client, m mqtt.Message) {
	var e domain.Sensor
	if err := json.Unmarshal(m.Payload(), &e); err != nil {
		log.Println("Invalid event:", err)
		return
	}

	processEvent(e)
}

func processEvent(e domain.Sensor) {
	log.Printf("Processed event: sensor=%s value=%.2f", e.Sensor, e.Value)
}
