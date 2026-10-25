package gateway

import (
	"errors"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
	"github.com/manaraph/stream-aggregator/pkg/monitoring"
	streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"
	"github.com/manaraph/stream-aggregator/pkg/ws"
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
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	go publishMetrics(hub, dispatcher)

	mux := http.NewServeMux()
	hub.RegisterRoute(mux)
	mux.Handle("/metrics", monitoring.Handler(dispatcher.MetricsSnapshot, dispatcher.LastReports))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("write health response: %v", err)
		}
	})

	httpServer := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	return &Gateway{Hub: hub, GrpcServer: grpcServer, HttpServer: httpServer}, nil
}

func publishMetrics(hub *ws.Hub, dispatcher grpcapi.Dispatcher) {
	startedAt := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	publishMetricsWithTicker(hub, dispatcher, ticker.C, startedAt)
}

func publishMetricsWithTicker(hub *ws.Hub, dispatcher grpcapi.Dispatcher, tick <-chan time.Time, startedAt time.Time) {
	var previousDelivered uint64

	for range tick {
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
				WebsocketDropped:  proto.Uint64(stats.Dropped),
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
