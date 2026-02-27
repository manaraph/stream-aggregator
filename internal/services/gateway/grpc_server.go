package gateway

import (
	"io"
	"log"
	"net"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
)

type sensorServer struct {
	streamv1.UnimplementedSensorServiceServer
}

func (s *sensorServer) IngestSensors(stream streamv1.SensorService_IngestSensorServer) error {
	for {
		e, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed stream")
			return stream.SendAndClose(&streamv1.IngestResponse{Success: true, Message: e.Sensor})
		}
		if err != nil {
			log.Println("Stream error:", err)
			return err
		}

		log.Printf("gRPC event: %+v", e)

		broadcast <- e
	}
}

func StartGRPC() {
	lis, _ := net.Listen("tcp", ":50051")
	grpcServer := grpc.NewServer()
	streamv1.RegisterSensorServiceServer(grpcServer, &sensorServer{})

	log.Println("gRPC streaming on :50051")
	grpcServer.Serve(lis)
}
