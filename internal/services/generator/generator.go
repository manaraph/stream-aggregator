package generator

import (
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/manaraph/stream-aggregator/pkg/broker"
)

func Start() {
	clientId := os.Getenv("GENERATOR_ID")
	if clientId == "" {
		log.Fatalf("GENERATOR_ID not defined")
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

	p := &Publisher{B: mclient}
	p.Run()
}
