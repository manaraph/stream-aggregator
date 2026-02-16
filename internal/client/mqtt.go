package client

import (
	"errors"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func InitMQTTClient(clientId string) (mqtt.Client, error) {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		return nil, errors.New("MQTT_BROKER not defined")
	}

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientId)
	mclient := mqtt.NewClient(opts)
	if token := mclient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return mclient, nil
}
