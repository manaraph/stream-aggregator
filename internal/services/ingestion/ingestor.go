package ingestion

import (
	"context"
	"errors"
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/grpc"

	"github.com/manaraph/stream-aggregator/internal/storage"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
)

type processorFactories struct {
	connectGateway       func() (streamv1.SensorServiceClient, *grpc.ClientConn, error)
	newBroker            func(string) (broker.Broker, error)
	newBrokerWithHandler func(string, mqtt.MessageHandler) (broker.Broker, error)
	openSensor           func(streamv1.SensorServiceClient, context.Context) (SensorStreamClient, error)
	openMetrics          func(*grpc.ClientConn, context.Context) (MetricsStreamClient, error)
	openStore            func(context.Context, string) (sensorStore, error)
}

func defaultProcessorFactories() processorFactories {
	return processorFactories{
		connectGateway: grpcapi.ConnectGateway,
		newBroker: func(id string) (broker.Broker, error) {
			return broker.NewMQTTClient(id)
		},
		newBrokerWithHandler: func(id string, handler mqtt.MessageHandler) (broker.Broker, error) {
			return broker.NewMQTTClientWithDefaultHandler(id, handler)
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

	stream, err := factories.openSensor(client, ctx)
	if err != nil {
		closeConn()
		return nil, fmt.Errorf("failed to open sensor gRPC stream: %w", err)
	}

	metricsStream, err := factories.openMetrics(conn, ctx)
	if err != nil {
		closeConn()
		return nil, fmt.Errorf("failed to open metrics gRPC stream: %w", err)
	}

	store, err := factories.openStore(ctx, databaseURL)
	if err != nil {
		closeConn()
		return nil, err
	}

	p := &Processor{
		store:      store,
		GRPC:       conn,
		S:          stream,
		M:          metricsStream,
		ctx:        ctx,
		eventQueue: make(chan queuedReading, positiveIntEnv("INGESTION_QUEUE_SIZE", defaultQueueSize)),
	}

	var mclient broker.Broker
	if factories.newBrokerWithHandler != nil {
		mclient, err = factories.newBrokerWithHandler(clientID, p.HandleMessage)
	} else {
		mclient, err = factories.newBroker(clientID)
	}

	if err != nil {
		store.Close()
		closeConn()
		return nil, err
	}

	p.B = mclient
	return p, nil
}
