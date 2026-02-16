package ingestion

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/manaraph/stream-aggregator/internal/domain"
	pb "github.com/manaraph/stream-aggregator/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var stream pb.SensorService_IngestSensorsClient

func init() {
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		addr = "gateway:50051"
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := pb.NewSensorServiceClient(conn)

	ctx := context.Background()
	s, err := client.IngestSensors(ctx)
	if err != nil {
		panic(err)
	}
	stream = s
}

func forwardEvent(data domain.Sensor) {
	t, err := time.Parse(time.RFC3339, data.Timestamp.Format(time.RFC3339))
	if err != nil {
		log.Println("Invalid timestamp:", data.Timestamp)
	}

	err = stream.Send(&pb.SensorData{
		Sensor:    data.Sensor,
		Value:     data.Value,
		Timestamp: timestamppb.New(t),
	})

	if err != nil {
		log.Println("gRPC stream send failed:", err)
		time.Sleep(3 * time.Second) // basic backoff
	}
}
