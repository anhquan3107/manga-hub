package fetcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mangahub/pkg/models"
)

// loadSeed reads and parses the existing manga seed file from disk.
func loadSeed(path string) ([]models.Manga, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read seed file: %w", err)
	}

	var mangaList []models.Manga
	if err := json.Unmarshal(data, &mangaList); err != nil {
		return nil, fmt.Errorf("parse seed file: %w", err)
	}

	return sanitizeMangaList(mangaList), nil
}

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

// pickLocalizedText retrieves a localized text value from attributes, trying multiple languages.
func pickLocalizedText(attributes map[string]any, key string) string {
	raw, ok := attributes[key]
	if !ok {
		return ""
	}

	localized, ok := raw.(map[string]any)
	if !ok {
		return ""
	}

	for _, lang := range []string{"en", "ja-ro", "ja"} {
		if value, ok := localized[lang].(string); ok && value != "" {
			return value
		}
	}

	for _, value := range localized {
		if text, ok := value.(string); ok && text != "" {
			return text
		}
	}

	return ""
}

// extractAuthor retrieves the author name from MangaDex relationships.
func extractAuthor(rels []struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}) string {
	for _, rel := range rels {
		if rel.Type != "author" {
			continue
		}
		if rel.Attributes == nil {
			continue
		}
		if name, ok := rel.Attributes["name"].(string); ok && name != "" {
			return name
		}
	}
	return ""
}

// extractTagNames retrieves tag names from MangaDex attributes.
func extractTagNames(attributes map[string]any) []string {
	rawTags, ok := attributes["tags"].([]any)
	if !ok {
		return nil
	}

	tags := make([]string, 0, len(rawTags))
	for _, rawTag := range rawTags {
		tagMap, ok := rawTag.(map[string]any)
		if !ok {
			continue
		}
		attr, ok := tagMap["attributes"].(map[string]any)
		if !ok {
			continue
		}
		name := pickLocalizedText(attr, "name")
		if name != "" {
			tags = append(tags, name)
		}
	}

	return tags
}

// uniqueNonEmpty removes duplicates and empty strings from a slice of strings.
func uniqueNonEmpty(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

// trimTo truncates a string to a maximum length, adding "..." if truncated.
func trimTo(input string, limit int) string {
	if len(input) <= limit {
		return input
	}
	if limit <= 3 {
		return input[:limit]
	}
	return input[:limit-3] + "..."
}

// writeJSON writes a data structure to a JSON file with indentation.
func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write json file: %w", err)
	}

	return nil
}

