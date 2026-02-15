package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	pb "github.com/manaraph/stream-aggregator/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Sensor struct {
	Sensor    string  `json:"sensor"`
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

func main() {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://mosquitto:1883"
	}

	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "gateway:50051"
	}

	var stream pb.SensorService_IngestSensorsClient
	var conn *grpc.ClientConn
	for {
		var err error
		conn, err = grpc.Dial(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println("Waiting for gateway gRPC server...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		client := pb.NewSensorServiceClient(conn)
		stream, err = client.IngestSensors(context.Background())
		if err != nil {
			log.Println("Failed to open gRPC stream, retrying...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Println("Connected to gRPC gateway at", gatewayAddr)
		break
	}
	defer conn.Close()

	sensorCh := make(chan *Sensor, 100)

	go func() {
		for sensor := range sensorCh {
			t, err := time.Parse(time.RFC3339, sensor.Timestamp)
			if err != nil {
				log.Println("Invalid timestamp:", sensor.Timestamp)
				continue
			}

			if err := stream.Send(&pb.SensorData{
				Sensor:    sensor.Sensor,
				Value:     sensor.Value,
				Timestamp: timestamppb.New(t),
			}); err != nil {
				log.Println("Failed to send to gRPC stream:", err)
			} else {
				log.Printf("Forwarded to Gateway: %s = %.2f", sensor.Sensor, sensor.Value)
			}
		}
	}()

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID("ingestion-service")
	mclient := mqtt.NewClient(opts)
	if token := mclient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	token := mclient.Subscribe("sensors/#", 0, func(c mqtt.Client, m mqtt.Message) {
		var sensor Sensor
		if err := json.Unmarshal(m.Payload(), &sensor); err != nil {
			log.Println("Failed to unmarshal MQTT message:", err)
			return
		}
		sensorCh <- &sensor
	})
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe to MQTT topic: %v", token.Error())
	}

	log.Println("Subscribed to MQTT topic sensors/#")
	select {}
}
