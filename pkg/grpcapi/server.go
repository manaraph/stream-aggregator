package grpcapi

import (
	"io"
	"log"

	"github.com/manaraph/stream-aggregator/pkg/events"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"google.golang.org/grpc"
)

type Server struct {
	streamv1.UnimplementedSensorServiceServer
	streamv1.UnimplementedMetricsServiceServer
	Dispatcher Dispatcher
}

func RegisterServices(grpcServer grpc.ServiceRegistrar, hub *ws.Hub) {
	dispatcher := NewWebSocketDispatcher(hub)
	server := &Server{Dispatcher: dispatcher}

	streamv1.RegisterSensorServiceServer(grpcServer, server)
	streamv1.RegisterMetricsServiceServer(grpcServer, server)
}

func (s *Server) IngestSensor(stream streamv1.SensorService_IngestSensorServer) error {
	for {
		e, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		msg := events.NewSensorMessage(e)
		s.Dispatcher.Publish(msg)
	}
}

func (s *Server) IngestMetrics(stream streamv1.MetricsService_IngestMetricsServer) error {
	for {
		e, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Println("Metrics stream closed:", err)
			return err
		}

		msg := events.NewMetricsMessage(e)
		s.Dispatcher.Publish(msg)
	}
}
