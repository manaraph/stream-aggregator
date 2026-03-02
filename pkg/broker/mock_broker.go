package broker

import mqtt "github.com/eclipse/paho.mqtt.golang"

type MockMessage struct {
	mqtt.Message
	PayloadData []byte
}
type FakeBroker struct {
	Messages chan []byte
}

func NewFakeBroker() *FakeBroker {
	return &FakeBroker{Messages: make(chan []byte, 10)}
}

func (f *FakeBroker) Publish(topic string, data []byte) error {
	f.Messages <- data
	return nil
}
func (f *FakeBroker) Subscribe(topic string, handler func(mqtt.Client, mqtt.Message)) error {
	go func() {
		for msg := range f.Messages {
			handler(nil, &MockMessage{PayloadData: msg})
		}
	}()
	return nil
}

func (f *FakeBroker) Close() error {
	close(f.Messages)
	return nil
}
