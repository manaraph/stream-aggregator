package ingestion

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/pkg/broker"
	"github.com/manaraph/stream-aggregator/pkg/grpcapi"
)

func Start() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clientId := os.Getenv("INGESTION_ID")
	if clientId == "" {
		log.Fatalf("INGESTION_ID not defined")
	}

	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		log.Fatalf("GATEWAY_ADDR not defined")
	}

	mbroker := os.Getenv("MQTT_BROKER")
	if mbroker == "" {
		log.Fatalf("MQTT_BROKER not defined")
	}

	opts := mqtt.NewClientOptions().AddBroker(mbroker).SetClientID(clientId)
	mc := mqtt.NewClient(opts)
	if token := mc.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("connection failed: ", token.Error())
	}

	mclient := broker.NewMQTTClient(mc)
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

	if err := p.Run(ctx); err != nil {
		log.Fatalf("Processor failed: %v", err)
	}

	<-ctx.Done()

	// Give the processor 5 seconds to drain the queue before killing it
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Close(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
}
