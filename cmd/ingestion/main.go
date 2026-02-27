package main

import "github.com/manaraph/stream-aggregator/internal/services/ingestion"

func main() {
	ingestion.StartPipeline()
	ingestion.Start()
}
