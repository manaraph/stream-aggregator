package ingestion

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	pb "github.com/manaraph/stream-aggregator/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	stream   pb.SensorService_IngestSensorsClient
	ssClient pb.SensorServiceClient
	mu       sync.Mutex
)

func connect() {
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		log.Fatalf("GATEWAY_ADDR not defined")
		return
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Println("gRPC dial failed:", err)
		time.Sleep(3 * time.Second)
		connect()
		return
	}

	ssClient = pb.NewSensorServiceClient(conn)
	openStream()
}

func openStream() {
	mu.Lock()
	defer mu.Unlock()

	ctx := context.Background()
	s, err := ssClient.IngestSensors(ctx)
	if err != nil {
		log.Println("Stream open failed:", err)
		time.Sleep(3 * time.Second)
		openStream()
		return
	}

	stream = s
	log.Println("gRPC stream connected")
}

func init() {
	connect()
}

func forwardEvent(data domain.Sensor) {
	mu.Lock()
	s := stream
	mu.Unlock()

	if s == nil {
		return
	}

	t, err := time.Parse(time.RFC3339, data.Timestamp.Format(time.RFC3339))
	if err != nil {
		log.Println("Invalid timestamp:", data.Timestamp)
	}

	err = s.Send(&pb.SensorData{
		Sensor:    data.Sensor,
		Value:     data.Value,
		Timestamp: timestamppb.New(t),
	})

	if err != nil {
		log.Println("gRPC send failed:", err)
		openStream() // reconnect
	}
}
