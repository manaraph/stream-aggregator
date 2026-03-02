package grpcapi

import (
	"net"
	"testing"

	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"google.golang.org/grpc"
)

func TestNewClient(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	s := grpc.NewServer()
	streamv1.RegisterSensorServiceServer(s, &Server{})
	go s.Serve(lis)
	defer s.Stop()

	client, conn, err := NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}

	conn.Close()
}
