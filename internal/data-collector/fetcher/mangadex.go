package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"mangahub/pkg/models"
)

const mangaDexBaseURL = "https://api.mangadex.org/manga"

// fetchMangaDexBatch fetches manga from MangaDex across multiple demographics.
func fetchMangaDexBatch(ctx context.Context, limit int) ([]models.Manga, error) {
	demographics := []string{"shounen", "shoujo", "seinen", "josei"}
	perGroup := 25
	if limit > 0 && limit/len(demographics) > 0 {
		perGroup = limit / len(demographics)
	}
	if perGroup <= 0 {
		perGroup = 25
	}

	items := make([]models.Manga, 0, limit)
	client := &http.Client{Timeout: 15 * time.Second}

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

// fetchMangaDexByDemographic fetches manga from MangaDex for a specific demographic.
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
