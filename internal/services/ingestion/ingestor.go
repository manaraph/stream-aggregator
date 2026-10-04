package ingestion

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/manaraph/stream-aggregator/internal/storage"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
)

type processorFactories struct {
	connectGateway func() (streamv1.SensorServiceClient, *grpc.ClientConn, error)
	newBroker      func(string) (broker.Broker, error)
	openSensor     func(streamv1.SensorServiceClient, context.Context) (SensorStreamClient, error)
	openMetrics    func(*grpc.ClientConn, context.Context) (MetricsStreamClient, error)
	openStore      func(context.Context, string) (sensorStore, error)
}

func defaultProcessorFactories() processorFactories {
	return processorFactories{
		connectGateway: grpcapi.ConnectGateway,
		newBroker: func(id string) (broker.Broker, error) {
			return broker.NewMQTTClient(id)
		},
		openSensor: func(client streamv1.SensorServiceClient, ctx context.Context) (SensorStreamClient, error) {
			return client.IngestSensor(ctx)
		},
		openMetrics: func(conn *grpc.ClientConn, ctx context.Context) (MetricsStreamClient, error) {
			return streamv1.NewMetricsServiceClient(conn).IngestMetrics(ctx)
		},
		openStore: func(ctx context.Context, url string) (sensorStore, error) {
			return storage.OpenPostgres(ctx, url)
		},
	}
}

func NewProcessor() (*Processor, error) {
	return newProcessor(
		context.Background(),
		os.Getenv("INGESTION_ID"),
		os.Getenv("DATABASE_URL"),
		defaultProcessorFactories(),
	)
}

func newProcessor(
	ctx context.Context,
	clientID, databaseURL string,
	factories processorFactories,
) (*Processor, error) {
	if clientID == "" {
		return nil, errors.New("INGESTION_ID not defined")
	}

	client, conn, err := factories.connectGateway()
	if err != nil {
		return nil, err
	}
	closeConn := func() {
		if conn != nil {
			_ = conn.Close()
		}
	}

	mclient, err := factories.newBroker(clientID)
	if err != nil {
		closeConn()
		return nil, err
	}
	closeBroker := func() { _ = mclient.Close() }

	stream, err := factories.openSensor(client, ctx)
	if err != nil {
		closeBroker()
		closeConn()
		return nil, fmt.Errorf("failed to open sensor gRPC stream: %w", err)
	}

	metricsStream, err := factories.openMetrics(conn, ctx)
	if err != nil {
		closeBroker()
		closeConn()
		return nil, fmt.Errorf("failed to open metrics gRPC stream: %w", err)
	}

	store, err := factories.openStore(ctx, databaseURL)
	if err != nil {
		closeBroker()
		closeConn()
		return nil, err
	}

	return &Processor{
		B:     mclient,
		store: store,
		GRPC:  conn,
		S:     stream,
		M:     metricsStream,
	}, nil
}
