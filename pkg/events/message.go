package events

import streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type MetricsData struct {
	Queue      *QueueMetricsData      `json:"queue,omitempty"`
	Throughput *ThroughputMetricsData `json:"throughput,omitempty"`
	Latency    *LatencyMetricsData    `json:"latency,omitempty"`
	Grpc       *ConnectionMetricsData `json:"grpc,omitempty"`
	Broker     *ConnectionMetricsData `json:"broker,omitempty"`
	Runtime    *RuntimeMetricsData    `json:"runtime,omitempty"`
}

type QueueMetricsData struct {
	Processed   uint64  `json:"processed"`
	Dropped     uint64  `json:"dropped"`
	Used        uint32  `json:"used"`
	Capacity    uint32  `json:"capacity"`
	MaxUsed     uint32  `json:"maxUsed"`
	Utilization float64 `json:"utilization"`
}

type ThroughputMetricsData struct {
	PublishRate   float64 `json:"publishRate"`
	IngestionRate float64 `json:"ingestionRate"`
	GrpcRate      float64 `json:"grpcRate"`
	WebsocketRate float64 `json:"websocketRate"`
}

type LatencyMetricsData struct {
	AverageMs uint32 `json:"averageMs"`
	P95Ms     uint32 `json:"p95Ms"`
	MaxMs     uint32 `json:"maxMs"`
}

type ConnectionMetricsData struct {
	Connected  bool   `json:"connected"`
	Reconnects uint32 `json:"reconnects"`
	Errors     uint32 `json:"errors"`
}

type RuntimeMetricsData struct {
	UptimeSeconds     uint64 `json:"uptimeSeconds"`
	Goroutines        uint32 `json:"goroutines"`
	MemoryBytes       uint64 `json:"memoryBytes"`
	WebsocketClients  uint32 `json:"websocketClients"`
	BackpressureLevel uint32 `json:"backpressureLevel"`
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
		Data: metricsData(m),
	}
}

func metricsData(m *streamv1.IngestMetricsRequest) MetricsData {
	data := MetricsData{}
	if queue := m.GetQueue(); queue != nil {
		data.Queue = &QueueMetricsData{
			Processed:   queue.GetProcessed(),
			Dropped:     queue.GetDropped(),
			Used:        queue.GetUsed(),
			Capacity:    queue.GetCapacity(),
			MaxUsed:     queue.GetMaxUsed(),
			Utilization: queue.GetUtilization(),
		}
	}

	if throughput := m.GetThroughput(); throughput != nil {
		data.Throughput = &ThroughputMetricsData{
			PublishRate:   throughput.GetPublishRate(),
			IngestionRate: throughput.GetIngestionRate(),
			GrpcRate:      throughput.GetGrpcRate(),
			WebsocketRate: throughput.GetWebsocketRate(),
		}
	}

	if latency := m.GetLatency(); latency != nil {
		data.Latency = &LatencyMetricsData{
			AverageMs: latency.GetAverageMs(),
			P95Ms:     latency.GetP95Ms(),
			MaxMs:     latency.GetMaxMs(),
		}
	}

	if grpc := m.GetGrpc(); grpc != nil {
		data.Grpc = connectionMetricsData(grpc)
	}

	if broker := m.GetBroker(); broker != nil {
		data.Broker = connectionMetricsData(broker)
	}

	if runtime := m.GetRuntime(); runtime != nil {
		data.Runtime = &RuntimeMetricsData{
			UptimeSeconds:     runtime.GetUptimeSeconds(),
			Goroutines:        runtime.GetGoroutines(),
			MemoryBytes:       runtime.GetMemoryBytes(),
			WebsocketClients:  runtime.GetWebsocketClients(),
			BackpressureLevel: runtime.GetBackpressureLevel(),
		}
	}

	return data
}

func connectionMetricsData(metrics *streamv1.ConnectionMetrics) *ConnectionMetricsData {
	return &ConnectionMetricsData{
		Connected:  metrics.GetConnected(),
		Reconnects: metrics.GetReconnects(),
		Errors:     metrics.GetErrors(),
	}
}
