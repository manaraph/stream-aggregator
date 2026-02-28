package grpcapi

import (
	"io"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
)

type Server struct {
	streamv1.UnimplementedSensorServiceServer
	Hub ws.Broadcaster
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
		s.Hub.BroadcastEvent(e)
	}
}
