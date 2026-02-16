package gateway

import (
	"io"
	"log"
	"net"

	pb "github.com/manaraph/stream-aggregator/proto"
	"google.golang.org/grpc"
)

type sensorServer struct {
	pb.UnimplementedSensorServiceServer
}

func (s *sensorServer) IngestSensors(stream pb.SensorService_IngestSensorsServer) error {
	for {
		e, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed stream")
			return stream.SendAndClose(&pb.IngestResponse{Success: true, Message: e.Sensor})
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
	pb.RegisterSensorServiceServer(grpcServer, &sensorServer{})

	log.Println("gRPC streaming on :50051")
	grpcServer.Serve(lis)
}
