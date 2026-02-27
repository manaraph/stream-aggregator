package broker

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

func (f *FakeBroker) Subscribe(topic string, handler func([]byte)) error {
	go func() {
		for msg := range f.Messages {
			handler(msg)
		}
	}()
	return nil
}

func (f *FakeBroker) Close() error {
	close(f.Messages)
	return nil
}
