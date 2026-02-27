package broker

type Broker interface {
	Publish(topic string, data []byte) error
	Subscribe(topic string, handler func([]byte)) error
	Close() error
}
