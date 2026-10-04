package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"

	ingestionmocks "github.com/manaraph/stream-aggregator/internal/services/ingestion/mocks"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
)

func factorySet(conn *grpc.ClientConn, b *broker.FakeBroker, store sensorStore) processorFactories {
	return processorFactories{
		connectGateway: func() (streamv1.SensorServiceClient, *grpc.ClientConn, error) { return nil, conn, nil },
		newBroker:      func(string) (broker.Broker, error) { return b, nil },
		openSensor: func(streamv1.SensorServiceClient, context.Context) (SensorStreamClient, error) {
			return &ingestionmocks.MockSensorStreamClient{}, nil
		},
		openMetrics: func(*grpc.ClientConn, context.Context) (MetricsStreamClient, error) {
			return &ingestionmocks.MockMetricsStreamClient{}, nil
		},
		openStore: func(context.Context, string) (sensorStore, error) { return store, nil },
	}
}

func TestNewProcessorRejectsMissingID(t *testing.T) {
	p, err := newProcessor(context.Background(), "", "", processorFactories{})
	if p != nil || err == nil || err.Error() != "INGESTION_ID not defined" {
		t.Fatalf("got processor=%v err=%v", p, err)
	}
}

func TestNewProcessorSuccess(t *testing.T) {
	conn, err := grpc.NewClient("passthrough:///constructor-test", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	store := NewMockSensorStore(t)
	store.EXPECT().Close()
	p, err := newProcessor(context.Background(), "test-ingestor", "postgres://unused", factorySet(conn, broker.NewFakeBroker(), store))
	if err != nil {
		t.Fatal(err)
	}
	if p.B == nil || p.store != store || p.GRPC != conn || p.S == nil || p.M == nil {
		t.Fatal("processor was not initialized with all dependencies")
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewProcessorFactoryFailuresCleanUp(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*processorFactories)
		wantError string
	}{
		{"gateway", func(f *processorFactories) {
			f.connectGateway = func() (streamv1.SensorServiceClient, *grpc.ClientConn, error) { return nil, nil, errors.New("gateway") }
		}, "gateway"},
		{"broker", func(f *processorFactories) {
			f.newBroker = func(string) (broker.Broker, error) { return nil, errors.New("broker") }
		}, "broker"},
		{"sensor stream", func(f *processorFactories) {
			f.openSensor = func(streamv1.SensorServiceClient, context.Context) (SensorStreamClient, error) {
				return nil, errors.New("sensor stream")
			}
		}, "failed to open sensor gRPC stream"},
		{"metrics stream", func(f *processorFactories) {
			f.openMetrics = func(*grpc.ClientConn, context.Context) (MetricsStreamClient, error) {
				return nil, errors.New("metrics stream")
			}
		}, "failed to open metrics gRPC stream"},
		{"store", func(f *processorFactories) {
			f.openStore = func(context.Context, string) (sensorStore, error) { return nil, errors.New("store") }
		}, "store"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := grpc.NewClient("passthrough:///constructor-failure", grpc.WithInsecure())
			if err != nil {
				t.Fatal(err)
			}
			b := broker.NewFakeBroker()
			f := factorySet(conn, b, NewMockSensorStore(t))
			tt.configure(&f)
			p, err := newProcessor(context.Background(), "test-ingestor", "", f)
			if p != nil || err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("got processor=%v err=%v", p, err)
			}
			if tt.name != "gateway" && conn.GetState().String() != "SHUTDOWN" {
				t.Fatalf("connection was not closed: %s", conn.GetState())
			}
		})
	}
}
