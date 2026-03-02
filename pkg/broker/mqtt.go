package broker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	mc mqtt.Client
}

func NewMQTTClient(mc mqtt.Client) *MQTTClient {
	return &MQTTClient{mc}
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
