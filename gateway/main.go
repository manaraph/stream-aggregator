package main

import (
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	pb "github.com/manaraph/stream-aggregator/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedSensorServiceServer
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *server) IngestSensors(stream pb.SensorService_IngestSensorsServer) error {
	for {
		data, err := stream.Recv()
		if err != nil {
			log.Println("Stream closed:", err)
			return err
		}

		s.mu.Lock()
		for client := range s.clients {
			client.WriteJSON(data)
		}
		s.mu.Unlock()
		log.Printf("Forwarded sensor: %s = %.2f", data.Sensor, data.Value)
	}
}

func main() {
	s := &server{clients: make(map[*websocket.Conn]bool)}

	// Start WebSocket server
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()
	})
	go func() {
		log.Println("WebSocket server listening on :8080")
		http.ListenAndServe(":8080", nil)
	}()

	// Start gRPC server
	lis, _ := net.Listen("tcp", ":50051")
	grpcServer := grpc.NewServer()
	pb.RegisterSensorServiceServer(grpcServer, s)
	log.Println("Gateway gRPC server listening on :50051")
	grpcServer.Serve(lis)
}
