package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/manaraph/stream-aggregator/internal/services/ingestion"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p, err := ingestion.NewProcessor()
	if err != nil {
		log.Println("connection failed: ", err)
		return
	}

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
