package events

import streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type MetricsMessage struct {
	Type string      `json:"type"`
	Data MetricsData `json:"data"`
}

type MetricsData struct {
	Processed     uint64  `json:"processed"`
	Dropped       uint64  `json:"dropped"`
	QueueUsed     uint32  `json:"queueUsed"`
	QueueCapacity uint32  `json:"queueCapacity"`
	QueuePercent  float64 `json:"queuePercent"`
	Rate          float64 `json:"rate"`
}

func NewSensorMessage(s *streamv1.IngestSensorRequest) Message {
	return Message{
		Type: "sensor",
		Data: s,
	}
}

func NewMetricsMessage(m *streamv1.StreamMetricsRequest) Message {
	return Message{
		Type: "metrics",
		Data: MetricsData{
			Processed:     m.GetProcessed(),
			Dropped:       m.GetDropped(),
			QueueUsed:     m.GetQueueUsed(),
			QueueCapacity: m.GetQueueCapacity(),
			QueuePercent:  m.GetQueuePercent(),
			Rate:          m.GetRate(),
		},
	}
}
