package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"mangahub/pkg/models"
)

const (
	defaultSeedFile = "manga.sample.json"
	mangaDexBaseURL = "https://api.mangadex.org/manga"
	jikanBaseURL    = "https://api.jikan.moe/v4/manga"
)

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

	client := &http.Client{Timeout: 15 * time.Second}
	fetched, source, err := fetchSeries(ctx, client, strings.TrimSpace(opts.Source), requested)
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
		FetchedCount:    len(fetched),
		ExistingCount:   len(existing),
		FinalCount:      len(merged),
	}, nil
}

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

func fetchSeries(ctx context.Context, client *http.Client, source string, limit int) ([]models.Manga, string, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "", "mangadex":
		items, err := fetchMangaDexBatch(ctx, client, limit)
		if err == nil {
			return sanitizeMangaList(items), "mangadex", nil
		}
		fallback, fallbackErr := fetchJikanBatch(ctx, client, limit)
		if fallbackErr != nil {
			return nil, "", fmt.Errorf("fetch mangadex: %v; fallback jikan: %w", err, fallbackErr)
		}
		return sanitizeMangaList(fallback), "jikan", nil
	case "jikan":
		items, err := fetchJikanBatch(ctx, client, limit)
		if err != nil {
			return nil, "", err
		}
		return sanitizeMangaList(items), "jikan", nil
	default:
		return nil, "", fmt.Errorf("unsupported source %q", source)
	}
}

func fetchMangaDexBatch(ctx context.Context, client *http.Client, limit int) ([]models.Manga, error) {
	demographics := []string{"shounen", "shoujo", "seinen", "josei"}
	perGroup := 25
	if limit > 0 && limit/len(demographics) > 0 {
		perGroup = limit / len(demographics)
	}
	if perGroup <= 0 {
		perGroup = 25
	}

	items := make([]models.Manga, 0, limit)
	for _, demographic := range demographics {
		batch, err := fetchMangaDexByDemographic(ctx, client, demographic, perGroup)
		if err != nil {
			return nil, err
		}
		items = append(items, batch...)
	}

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func fetchJikanBatch(ctx context.Context, client *http.Client, total int) ([]models.Manga, error) {
	if total <= 0 {
		total = 100
	}

	pages := total / 25
	if total%25 != 0 {
		pages++
	}

	items := make([]models.Manga, 0, total)
	for page := 1; page <= pages; page++ {
		batch, err := fetchJikanPage(ctx, client, page, 25)
		if err != nil {
			return nil, err
		}
		items = append(items, batch...)
	}

	if len(items) > total {
		items = items[:total]
	}

	return items, nil
}

func fetchJikanPage(ctx context.Context, client *http.Client, page, limit int) ([]models.Manga, error) {
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("order_by", "members")
	query.Set("sort", "desc")
	query.Set("sfw", "true")

	endpoint := jikanBaseURL + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build jikan request: %w", err)
	}
	req.Header.Set("User-Agent", "mangahub-importer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call jikan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jikan status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded jikanResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode jikan payload: %w", err)
	}

	results := make([]models.Manga, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = strings.TrimSpace(item.TitleEnglish)
		}
		if title == "" {
			continue
		}

		author := "Unknown"
		if len(item.Authors) > 0 {
			author = strings.TrimSpace(item.Authors[0].Name)
			if author == "" {
				author = "Unknown"
			}
		}

		description := strings.TrimSpace(item.Synopsis)
		if description == "" {
			description = "No description from API."
		}

		genres := make([]string, 0, len(item.Genres)+len(item.Demographics))
		for _, g := range item.Genres {
			name := strings.TrimSpace(g.Name)
			if name != "" {
				genres = append(genres, name)
			}
		}
		for _, d := range item.Demographics {
			name := strings.TrimSpace(d.Name)
			if name != "" {
				genres = append(genres, name)
			}
		}

		status := strings.ToLower(strings.TrimSpace(item.Status))
		if status == "" {
			status = "ongoing"
		}

		results = append(results, models.Manga{
			ID:            strconv.Itoa(item.MalID),
			Title:         title,
			Author:        author,
			Genres:        uniqueNonEmpty(genres),
			Status:        status,
			TotalChapters: item.Chapters,
			Description:   trimTo(description, 280),
			CoverURL:      "",
		})
	}

	return results, nil
}

func fetchMangaDexByDemographic(ctx context.Context, client *http.Client, demographic string, limit int) ([]models.Manga, error) {
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("contentRating[]", "safe")
	query.Set("includes[]", "author")
	query.Set("order[followedCount]", "desc")
	query.Set("availableTranslatedLanguage[]", "en")
	query.Set("publicationDemographic[]", demographic)

	endpoint := mangaDexBaseURL + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build mangadex request: %w", err)
	}
	req.Header.Set("User-Agent", "mangahub-importer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call mangadex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("mangadex status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded mangadexResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode mangadex payload: %w", err)
	}

	results := make([]models.Manga, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		title := pickLocalizedText(item.Attributes, "title")
		if title == "" {
			continue
		}

		description := pickLocalizedText(item.Attributes, "description")
		if description == "" {
			description = "No description from API."
		}

		author := extractAuthor(item.Relationships)
		if author == "" {
			author = "Unknown"
		}

		status, _ := item.Attributes["status"].(string)
		if status == "" {
			status = "ongoing"
		}

		tags := extractTagNames(item.Attributes)
		tags = append(tags, cases.Title(language.English).String(demographic))

		results = append(results, models.Manga{
			ID:            item.ID,
			Title:         title,
			Author:        author,
			Genres:        uniqueNonEmpty(tags),
			Status:        strings.ToLower(status),
			TotalChapters: 0,
			Description:   trimTo(description, 280),
			CoverURL:      "",
		})
	}

	return results, nil
}

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

func trimTo(input string, limit int) string {
	if len(input) <= limit {
		return input
	}
	if limit <= 3 {
		return input[:limit]
	}
	return input[:limit-3] + "..."
}
