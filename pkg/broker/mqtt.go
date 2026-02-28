package broker

import (
	"errors"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	mc mqtt.Client
}

func NewMQTTClient(clientId string) (*MQTTClient, error) {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		return nil, errors.New("MQTT_BROKER not defined")
	}

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientId)
	mc := mqtt.NewClient(opts)
	if token := mc.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTClient{mc}, nil
}

func (c *MQTTClient) Publish(topic string, data []byte) error {
	token := c.mc.Publish(topic, 0, false, data)
	return token.Error()
}

func (c *MQTTClient) Subscribe(topic string, handler func(c mqtt.Client, m mqtt.Message)) error {
	token := c.mc.Subscribe(topic, 0, handler)
	return token.Error()
}

func (c *MQTTClient) Close() error {
	if c.mc != nil {
		c.mc.Disconnect(250)
	}
	return nil
}
