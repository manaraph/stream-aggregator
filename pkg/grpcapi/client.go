package grpcapi

import (
	"errors"
	"os"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewClient(addr string) (streamv1.SensorServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return streamv1.NewSensorServiceClient(conn), conn, nil
}

func ConnectGateway() (
	streamv1.SensorServiceClient,
	*grpc.ClientConn,
	error,
) {
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		return nil, nil, errors.New("GATEWAY_ADDR not defined")
	}

	return NewClient(addr)
}
