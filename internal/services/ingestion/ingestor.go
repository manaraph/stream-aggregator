package ingestion

import (
	"context"
	"log"
	"os"

	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
)

func Start() {
	clientId := os.Getenv("INGESTION_ID")
	if clientId == "" {
		log.Fatalf("INGESTION_ID not defined")
	}

	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		log.Fatalf("GATEWAY_ADDR not defined")
	}

	mclient, err := broker.NewMQTTClient(clientId)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mclient.Close()

	client, conn, err := grpcapi.NewClient(addr)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC gateway: %v", err)
	}
	defer conn.Close()

	stream, err := client.IngestSensor(context.Background())
	if err != nil {
		log.Fatalf("Failed to open gRPC stream: %v", err)
	}

	p := &Processor{B: mclient, S: stream}

	if err := p.Run(); err != nil {
		log.Fatalf("Processor failed: %v", err)
	}

	select {}
}
