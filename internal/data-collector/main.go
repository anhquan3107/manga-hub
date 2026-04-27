package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mangahub/pkg/models"
)

const (
	manualFileDefault = "./data/manga.manual.json"
	outputFileDefault = "./data/manga.sample.json"
	reportFileDefault = "./data/collection_report.json"
	mangaDexBaseURL   = "https://api.mangadex.org/manga"
	jikanBaseURL      = "https://api.jikan.moe/v4/manga"
)

type mangadexResponse struct {
	Data []struct {
		ID            string         `json:"id"`
		Type          string         `json:"type"`
		Attributes    map[string]any `json:"attributes"`
		Relationships []struct {
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

type jikanResponse struct {
	Data []struct {
		MalID        int    `json:"mal_id"`
		URL          string `json:"url"`
		Status       string `json:"status"`
		Chapters     int    `json:"chapters"`
		Synopsis     string `json:"synopsis"`
		Title        string `json:"title"`
		TitleEnglish string `json:"title_english"`
		Authors      []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Genres []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Demographics []struct {
			Name string `json:"name"`
		} `json:"demographics"`
	} `json:"data"`
}

type collectionReport struct {
	GeneratedAt         string         `json:"generated_at"`
	ManualCount         int            `json:"manual_count"`
	APICount            int            `json:"api_count"`
	APISource           string         `json:"api_source"`
	MangaDexCount       int            `json:"mangadex_count"`
	FinalCount          int            `json:"final_count"`
	InvalidManualCount  int            `json:"invalid_manual_count"`
	InvalidAPICount     int            `json:"invalid_api_count"`
	DemographicCounters map[string]int `json:"demographic_counters"`
	EducationalPractice struct {
		QuotesCount int    `json:"quotes_count"`
		HTTPBinURL  string `json:"httpbin_url"`
	} `json:"educational_practice"`
}

type quotesResponse struct {
	Quotes []struct {
		Quote  string `json:"quote"`
		Author string `json:"author"`
	} `json:"quotes"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	manualPath := envOr("MANUAL_FILE", manualFileDefault)
	outputPath := envOr("OUTPUT_FILE", outputFileDefault)
	reportPath := envOr("REPORT_FILE", reportFileDefault)

	manualItems, err := loadManualManga(manualPath)
	if err != nil {
		log.Fatalf("load manual manga: %v", err)
	}
	manualItems, invalidManual := sanitizeMangaList(manualItems)
	if len(manualItems) < 20 {
		log.Fatalf("manual manga entries too low after validation: got=%d required>=20", len(manualItems))
	}

	client := &http.Client{Timeout: 15 * time.Second}
	apiItems, apiSource, err := fetchMangaDexBatch(ctx, client)
	if err != nil {
		log.Fatalf("fetch mangadex data: %v", err)
	}
	apiItems, invalidAPI := sanitizeMangaList(apiItems)

	quotesCount, httpbinURL := runEducationalPractice(ctx, client)

	merged := mergeAndDedupe(manualItems, apiItems)
	sort.Slice(merged, func(i, j int) bool {
		return strings.ToLower(merged[i].Title) < strings.ToLower(merged[j].Title)
	})

	if err := writeJSON(outputPath, merged); err != nil {
		log.Fatalf("write output json: %v", err)
	}

	report := buildReport(manualItems, apiItems, merged, quotesCount, httpbinURL, apiSource, invalidManual, invalidAPI)
	if err := writeJSON(reportPath, report); err != nil {
		log.Fatalf("write report json: %v", err)
	}

	log.Printf("collection complete: manual=%d api=%d merged=%d source=%s invalid_manual=%d invalid_api=%d", len(manualItems), len(apiItems), len(merged), apiSource, invalidManual, invalidAPI)
	log.Printf("output: %s", outputPath)
	log.Printf("report: %s", reportPath)
}

func loadManualManga(path string) ([]models.Manga, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manual file: %w", err)
	}

	type manualManga struct {
		Title         string   `json:"title"`
		Author        string   `json:"author"`
		Genres        []string `json:"genres"`
		Status        string   `json:"status"`
		TotalChapters int      `json:"total_chapters"`
		Description   string   `json:"description"`
		CoverURL      string   `json:"cover_url"`
	}

	var raw []manualManga
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("decode manual file: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("manual file has no manga entries")
	}

	items := make([]models.Manga, 0, len(raw))
	for _, item := range raw {
		items = append(items, models.Manga{
			Title:         item.Title,
			Author:        item.Author,
			Genres:        item.Genres,
			Status:        item.Status,
			TotalChapters: item.TotalChapters,
			Description:   item.Description,
			CoverURL:      item.CoverURL,
		})
	}

	for i := range items {
		if items[i].Status == "" {
			items[i].Status = "ongoing"
		}
	}

	return items, nil
}

func fetchMangaDexBatch(ctx context.Context, client *http.Client) ([]models.Manga, string, error) {
	demographics := []string{"shounen", "shoujo", "seinen", "josei"}
	items := make([]models.Manga, 0, 120)

	for _, demographic := range demographics {
		batch, err := fetchMangaDexByDemographic(ctx, client, demographic, 25)
		if err != nil {
			log.Printf("mangadex unavailable (%v), falling back to jikan", err)
			fallback, fallbackErr := fetchJikanBatch(ctx, client, 100)
			if fallbackErr != nil {
				return nil, "", fallbackErr
			}
			return fallback, "jikan", nil
		}
		items = append(items, batch...)
	}

	return items, "mangadex", nil
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
	req.Header.Set("User-Agent", "mangahub-data-collector/1.0")

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
	req.Header.Set("User-Agent", "mangahub-data-collector/1.0")

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
		tags = append(tags, strings.Title(demographic))

		results = append(results, models.Manga{
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

func runEducationalPractice(ctx context.Context, client *http.Client) (int, string) {
	quotesCount := 0
	httpbinURL := ""

	qReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://quotes.toscrape.com/api/quotes?page=1", nil)
	qResp, qErr := client.Do(qReq)
	if qErr == nil {
		defer qResp.Body.Close()
		if qResp.StatusCode < 300 {
			var quotes quotesResponse
			if err := json.NewDecoder(qResp.Body).Decode(&quotes); err == nil {
				quotesCount = len(quotes.Quotes)
			}
		}
	}

	hReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://httpbin.org/get", nil)
	hResp, hErr := client.Do(hReq)
	if hErr == nil {
		defer hResp.Body.Close()
		if hResp.StatusCode < 300 {
			var payload map[string]any
			if err := json.NewDecoder(hResp.Body).Decode(&payload); err == nil {
				if got, ok := payload["url"].(string); ok {
					httpbinURL = got
				}
			}
		}
	}

	return quotesCount, httpbinURL
}

func buildReport(manual, api, merged []models.Manga, quotesCount int, httpbinURL, apiSource string, invalidManual, invalidAPI int) collectionReport {
	counters := map[string]int{
		"shounen": 0,
		"shoujo":  0,
		"seinen":  0,
		"josei":   0,
	}

	for _, m := range merged {
		for _, g := range m.Genres {
			switch strings.ToLower(g) {
			case "shounen":
				counters["shounen"]++
			case "shoujo":
				counters["shoujo"]++
			case "seinen":
				counters["seinen"]++
			case "josei":
				counters["josei"]++
			}
		}
	}

	report := collectionReport{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		ManualCount:         len(manual),
		APICount:            len(api),
		APISource:           apiSource,
		MangaDexCount:       len(api),
		FinalCount:          len(merged),
		InvalidManualCount:  invalidManual,
		InvalidAPICount:     invalidAPI,
		DemographicCounters: counters,
	}
	report.EducationalPractice.QuotesCount = quotesCount
	report.EducationalPractice.HTTPBinURL = httpbinURL

	return report
}

func sanitizeMangaList(items []models.Manga) ([]models.Manga, int) {
	out := make([]models.Manga, 0, len(items))
	invalid := 0

	for _, item := range items {
		normalized, ok := normalizeManga(item)
		if !ok {
			invalid++
			continue
		}
		out = append(out, normalized)
	}

	return out, invalid
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

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer(
		" ", "-",
		"_", "-",
		"'", "",
		"\"", "",
		".", "",
		",", "",
		":", "",
		";", "",
		"/", "-",
		"&", "and",
		"(", "",
		")", "",
	)
	return replacer.Replace(input)
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

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
