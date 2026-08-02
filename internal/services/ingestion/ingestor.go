package ingestion

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
)

func NewProcessor() (*Processor, error) {
	clientId := os.Getenv("INGESTION_ID")
	if clientId == "" {
		return nil, errors.New("INGESTION_ID not defined")
	}

	client, conn, err := grpcapi.ConnectGateway()
	if err != nil {
		return nil, err
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	ctx := context.Background()
	stream, err := client.IngestSensor(ctx)
	if err != nil {
		_ = mclient.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("Failed to open gRPC stream: %w", err)
	}

	metricsStream, err := streamv1.NewMetricsServiceClient(conn).IngestMetrics(ctx)
	if err != nil {
		_ = mclient.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("Failed to open metrics stream: %w", err)
	}

	return &Processor{B: mclient, GRPC: conn, S: stream, M: metricsStream}, nil
}
