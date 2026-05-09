package main

import (
	"os"
	"strconv"
	"strings"

	fetcher "mangahub/internal/data-collector/fetcher"
)

const (
	outputFileDefault = "./data/manga.sample.json"
)

func sourcesOptions(seed, source string, limit int) fetcher.Options {
	return fetcher.Options{
		SeedFile: seed,
		Source:   source,
		Limit:    limit,
	}
}

// envOr reads a string from environment or returns the fallback default.
func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// envOrInt reads an integer from environment or returns the fallback default.
func envOrInt(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
