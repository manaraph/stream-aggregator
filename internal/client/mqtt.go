package client

import (
	"errors"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

func InitMQTTClient(clientId string) (*mqtt.Client, error) {
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("No .env file found, reading from env vars")
	}

	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://mosquitto:1883"
	}

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientId)
	mclient := mqtt.NewClient(opts)
	if token := mclient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &mclient, nil
}
