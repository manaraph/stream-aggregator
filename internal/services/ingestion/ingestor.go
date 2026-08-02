package ingestion

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
)

func NewProcessor() (*Processor, error) {
	clientId := os.Getenv("INGESTION_ID")
	if clientId == "" {
		return nil, errors.New("INGESTION_ID not defined")
	}

	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		return nil, errors.New("GATEWAY_ADDR not defined")
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		return nil, err
	}

	client, conn, err := grpcapi.NewClient(addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to gRPC gateway: %w", err)

	}

	ctx := context.Background()
	stream, err := client.IngestSensor(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Failed to open gRPC stream: %w", err)
	}

	metricsStream, err := client.StreamMetrics(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Failed to open metrics stream: %w", err)
	}

	return &Processor{B: mclient, GRPC: conn, S: stream, M: metricsStream}, nil
}
