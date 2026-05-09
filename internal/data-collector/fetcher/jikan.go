package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mangahub/pkg/models"
)

const jikanBaseURL = "https://api.jikan.moe/v4/manga"

// fetchJikanBatch fetches manga from Jikan API in paginated batches.
func fetchJikanBatch(ctx context.Context, total int) ([]models.Manga, error) {
	if total <= 0 {
		total = 100
	}

	pages := total / 25
	if total%25 != 0 {
		pages++
	}

	items := make([]models.Manga, 0, total)
	client := &http.Client{Timeout: 15 * time.Second}

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

// fetchJikanPage fetches a single page of manga from Jikan API.
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
