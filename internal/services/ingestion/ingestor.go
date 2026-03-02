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

	stream, err := client.IngestSensor(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Failed to open gRPC stream: %w", err)
	}

	return &Processor{B: mclient, GRPC: conn, S: stream}, nil
}
