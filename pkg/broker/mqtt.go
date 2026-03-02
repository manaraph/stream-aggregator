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
	mbroker := os.Getenv("MQTT_BROKER")
	if mbroker == "" {
		return nil, errors.New("MQTT_BROKER not defined")
	}

	opts := mqtt.NewClientOptions().AddBroker(mbroker).SetClientID(clientId)
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
	if handler == nil {
		return errors.New("mqtt: handler cannot be nil")
	}

	token := c.mc.Subscribe(topic, 0, handler)
	return token.Error()
}

func (c *MQTTClient) Close() error {
	if c.mc != nil {
		c.mc.Disconnect(250)
	}
	return nil
}
