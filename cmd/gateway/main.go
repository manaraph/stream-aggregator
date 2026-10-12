package main

import (
	"log"
	"os"

	"github.com/manaraph/stream-aggregator/internal/services/gateway"
)

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		log.Fatalf("GRPC_PORT not defined")
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		log.Fatalf("HTTP_PORT not defined")
	}

	if _, err := gateway.NewGateway(grpcPort, httpPort); err != nil {
		log.Fatalf("failed to start gateway: %v", err)
	}

	log.Println("gRPC streaming on", grpcPort)
	log.Println("HTTP on", httpPort)

	select {}
}
