package events

import streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func NewSensorMessage(s *streamv1.IngestSensorRequest) Message {
	return Message{
		Type: "sensor",
		Data: s,
	}
}

func NewMetricsMessage(m *streamv1.IngestMetricsRequest) Message {
	return Message{
		Type: "metrics",
		Data: m,
	}
}
