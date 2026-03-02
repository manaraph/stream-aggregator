package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manaraph/stream-aggregator/internal/services/generator"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p, err := generator.NewPublisherFromEnv()
	if err != nil {
		log.Println("connection failed: ", err)
		return
	}

	p.Run(ctx)
}
