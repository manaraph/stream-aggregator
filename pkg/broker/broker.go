package broker

import mqtt "github.com/eclipse/paho.mqtt.golang"

type Broker interface {
	Publish(topic string, data []byte) error
	// TODO: consider Subscribe(topic string, handler func([]byte)) error
	Subscribe(topic string, handler func(mqtt.Client, mqtt.Message)) error
	Close() error
}
