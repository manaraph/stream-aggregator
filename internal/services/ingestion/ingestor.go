package ingestion

import (
	"context"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

	if err := p.Run(); err != nil {
		log.Fatalf("Processor failed: %v", err)
	}

	select {}
}
