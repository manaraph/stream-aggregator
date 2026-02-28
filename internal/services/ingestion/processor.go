package ingestion

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StreamClient interface {
	Send(*streamv1.SensorEventRequest) error
}

type Processor struct {
	B          broker.Broker
	S          StreamClient
	eventQueue chan domain.Sensor
	processed  uint64
	dropped    uint64
}

func (p *Processor) Run() error {
	p.initPipeline()
	return p.B.Subscribe("sensors/#", p.HandleMessage)
}

func (p *Processor) HandleMessage(c mqtt.Client, m mqtt.Message) {
	var e domain.Sensor
	if err := json.Unmarshal(m.Payload(), &e); err != nil {
		log.Println("Invalid event:", err)
		return
	}
	p.EnqueueEvent(e)
}

func (p *Processor) ForwardEvent(data domain.Sensor) {
	err := p.S.Send(&streamv1.SensorEventRequest{
		Sensor:    data.Sensor,
		Value:     data.Value,
		Timestamp: timestamppb.New(data.Timestamp),
	})

	if err != nil {
		log.Println("gRPC send failed:", err)
	}
}
