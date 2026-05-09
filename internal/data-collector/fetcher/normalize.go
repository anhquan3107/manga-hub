package fetcher

import (
	"strings"

	"mangahub/pkg/models"
)

// sanitizeMangaList validates and normalizes a list of manga entries.
func sanitizeMangaList(items []models.Manga) []models.Manga {
	out := make([]models.Manga, 0, len(items))
	for _, item := range items {
		normalized, ok := normalizeManga(item)
		if !ok {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

// normalizeManga validates and normalizes a single manga entry.
// Returns false if the entry fails validation (missing required fields or invalid data).
func normalizeManga(item models.Manga) (models.Manga, bool) {
	item.Title = strings.TrimSpace(item.Title)
	item.Author = strings.TrimSpace(item.Author)
	item.Description = strings.TrimSpace(item.Description)
	item.CoverURL = strings.TrimSpace(item.CoverURL)

	// Validate required fields
	if item.Title == "" || item.Author == "" || item.Description == "" {
		return models.Manga{}, false
	}

	item.Genres = uniqueNonEmpty(item.Genres)
	if len(item.Genres) == 0 {
		return models.Manga{}, false
	}

	if item.TotalChapters < 0 {
		item.TotalChapters = 0
	}

	status := strings.ToLower(strings.TrimSpace(item.Status))
	switch status {
	case "ongoing", "completed", "hiatus", "cancelled":
		item.Status = status
	default:
		item.Status = "ongoing"
	}

	return item, true
}

// mergeAndDedupe combines two manga lists and removes duplicates by title.
func mergeAndDedupe(a, b []models.Manga) []models.Manga {
	seen := make(map[string]struct{}, len(a)+len(b))
	merged := make([]models.Manga, 0, len(a)+len(b))

	add := func(item models.Manga) {
		key := strings.ToLower(strings.TrimSpace(item.Title))
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, item)
	}

	for _, item := range a {
		add(item)
	}
	for _, item := range b {
		add(item)
	}

	return merged
}
