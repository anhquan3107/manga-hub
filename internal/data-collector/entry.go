package main

import (
	"context"
	"log"
	"time"

	fetcher "mangahub/internal/data-collector/fetcher"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// build options from environment (keeps previous env names for compatibility)
	seedFile := envOr("OUTPUT_FILE", outputFileDefault)
	src := envOr("API_SOURCE", "")
	limit := envOrInt("IMPORT_LIMIT", 0)

	opts := sourcesOptions(seedFile, src, limit)

	res, err := fetcher.Collect(ctx, opts)
	if err != nil {
		log.Fatalf("collect failed: %v", err)
	}

	log.Printf("collection complete: source=%s requested=%d fetched=%d existing=%d final=%d", res.Source, res.RequestedLimit, res.FetchedCount, res.ExistingCount, res.FinalCount)
	log.Printf("output: %s", seedFile)
}
