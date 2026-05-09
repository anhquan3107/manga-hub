package fetcher

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mangahub/pkg/models"
)

const defaultSeedFile = "manga.sample.json"

// Collect is the public API that orchestrates the entire collection workflow.
// It loads existing manga data, fetches new data from the specified source,
// merges and deduplicates them, and writes the result back to the seed file.
func Collect(ctx context.Context, opts Options) (Result, error) {
	seedFile := strings.TrimSpace(opts.SeedFile)
	if seedFile == "" {
		seedFile = defaultSeedFile
	}

	requested := opts.Limit
	if requested <= 0 {
		requested = 100
	}

	existing, err := loadSeed(seedFile)
	if err != nil {
		return Result{}, err
	}

	// HTTP client is created internally for all fetches
	fetched, source, err := fetchSeries(ctx, strings.TrimSpace(opts.Source), requested)
	if err != nil {
		return Result{}, err
	}

	merged := mergeAndDedupe(existing, fetched)
	sort.Slice(merged, func(i, j int) bool {
		return strings.ToLower(merged[i].Title) < strings.ToLower(merged[j].Title)
	})

	if err := writeJSON(seedFile, merged); err != nil {
		return Result{}, err
	}

	return Result{
		Source:         source,
		RequestedLimit: requested,
		FetchedCount:   len(fetched),
		ExistingCount:  len(existing),
		FinalCount:     len(merged),
	}, nil
}

// fetchSeries determines the source and delegates to the appropriate fetcher.
// It supports "mangadex" (with "jikan" as fallback) and "jikan" as sources.
func fetchSeries(ctx context.Context, source string, limit int) ([]models.Manga, string, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "", "mangadex":
		items, err := fetchMangaDexBatch(ctx, limit)
		if err == nil {
			return sanitizeMangaList(items), "mangadex", nil
		}
		// Fallback to Jikan if MangaDex fails
		fallback, fallbackErr := fetchJikanBatch(ctx, limit)
		if fallbackErr != nil {
			return nil, "", fmt.Errorf("fetch mangadex: %v; fallback jikan: %w", err, fallbackErr)
		}
		return sanitizeMangaList(fallback), "jikan", nil
	case "jikan":
		items, err := fetchJikanBatch(ctx, limit)
		if err != nil {
			return nil, "", err
		}
		return sanitizeMangaList(items), "jikan", nil
	default:
		return nil, "", fmt.Errorf("unsupported source %q", source)
	}
}
