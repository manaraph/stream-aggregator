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
	return newMQTTClient(clientId, nil)
}

// NewMQTTClientWithDefaultHandler installs the handler before connecting. This
// matters for persistent sessions: a broker can deliver queued QoS messages as
// soon as the connection is established, before Subscribe registers a route.
func NewMQTTClientWithDefaultHandler(clientId string, handler mqtt.MessageHandler) (*MQTTClient, error) {
	return newMQTTClient(clientId, handler)
}

func newMQTTClient(clientId string, handler mqtt.MessageHandler) (*MQTTClient, error) {
	mbroker := os.Getenv("MQTT_BROKER")
	if mbroker == "" {
		return nil, errors.New("MQTT_BROKER not defined")
	}

	opts := mqtt.NewClientOptions().AddBroker(mbroker).
		SetClientID(clientId).
		SetCleanSession(false).
		SetOrderMatters(false).
		SetAutoAckDisabled(true)
	if handler != nil {
		opts.SetDefaultPublishHandler(handler)
	}
	mc := mqtt.NewClient(opts)
	if token := mc.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTClient{mc}, nil
}

// Connected reports the live MQTT connection state rather than merely
// indicating that a client object was constructed.
func (c *MQTTClient) Connected() bool {
	return c != nil && c.mc != nil && c.mc.IsConnected()
}

func (c *MQTTClient) Publish(topic string, data []byte) error {
	token := c.mc.Publish(topic, 1, false, data)
	token.Wait()
	return token.Error()
}

func (c *MQTTClient) Subscribe(topic string, handler func(c mqtt.Client, m mqtt.Message)) error {
	if handler == nil {
		return errors.New("mqtt: handler cannot be nil")
	}

	token := c.mc.Subscribe(topic, 1, handler)
	token.Wait()
	return token.Error()
}

func (c *MQTTClient) Close() error {
	if c.mc != nil {
		c.mc.Disconnect(250)
	}
	return nil
}
