package main

import "github.com/manaraph/stream-aggregator/internal/services/gateway"

func main() {
	go gateway.StartBroadcastLoop()
	go gateway.StartGRPC()
	gateway.StartHTTP()
}
