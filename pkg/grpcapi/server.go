package grpcapi

import (
	"io"
	"log"

	"github.com/manaraph/stream-aggregator/pkg/events"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
)

type Server struct {
	streamv1.UnimplementedSensorServiceServer
	Dispatcher Dispatcher
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

func (s *Server) StreamMetrics(stream streamv1.SensorService_StreamMetricsServer) error {
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
