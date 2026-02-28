package gateway

import (
	"net"
	"net/http"

	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"google.golang.org/grpc"
)

type Gateway struct {
	Hub        *ws.Hub
	GrpcServer *grpc.Server
	HttpServer *http.Server
}

func NewGateway(grpcAddr, httpAddr string) (*Gateway, error) {
	hub := ws.NewHub()

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	streamv1.RegisterSensorServiceServer(grpcServer, &grpcapi.Server{Hub: hub})

	go hub.Run()
	go grpcServer.Serve(lis)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.Handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	httpServer := &http.Server{Addr: httpAddr, Handler: mux}
	go httpServer.ListenAndServe()

	return &Gateway{Hub: hub, GrpcServer: grpcServer, HttpServer: httpServer}, nil
}
