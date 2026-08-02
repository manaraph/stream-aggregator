package gateway

import (
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
	dispatcher := grpcapi.NewWebSocketDispatcher(hub)
	grpcapi.RegisterServices(grpcServer, dispatcher)

	go hub.Run()
	go grpcServer.Serve(lis)
	go publishMetrics(hub, dispatcher)

	mux := http.NewServeMux()
	hub.RegisterRoute(mux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	httpServer := &http.Server{Addr: httpAddr, Handler: mux}
	go httpServer.ListenAndServe()

	return &Gateway{Hub: hub, GrpcServer: grpcServer, HttpServer: httpServer}, nil
}

func publishMetrics(hub *ws.Hub, dispatcher grpcapi.Dispatcher) {
	startedAt := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var previousDelivered uint64

	for range ticker.C {
		stats := hub.Stats()
		delivered := stats.Delivered
		dispatcher.PublishMetrics(&streamv1.IngestMetricsRequest{
			Throughput: &streamv1.ThroughputMetrics{WebsocketRate: proto.Float64(float64(delivered-previousDelivered) / 5)},
			Runtime: &streamv1.RuntimeMetrics{
				UptimeSeconds:     proto.Uint64(uint64(time.Since(startedAt).Seconds())),
				Goroutines:        proto.Uint32(uint32(runtime.NumGoroutine())),
				MemoryBytes:       proto.Uint64(allocatedMemory()),
				WebsocketClients:  proto.Uint32(stats.Clients),
				BackpressureLevel: proto.Uint32(stats.BackpressureLevel),
			},
		})
		previousDelivered = delivered
	}
}

func allocatedMemory() uint64 {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return memory.Alloc
}
