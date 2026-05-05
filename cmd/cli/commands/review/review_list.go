package review

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

type reviewListItem struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Rating    int    `json:"rating"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
	Helpful   int    `json:"helpful"`
}

func reviewList(args []string) {
	var mangaID string
	var limit int
	var sortBy string
	flags := flag.NewFlagSet("review list", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga")
	flags.IntVar(&limit, "limit", 50, "Max reviews (1-200)")
	flags.StringVar(&sortBy, "sort", "recent", "Sort by (recent, helpful)")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}
	if sortBy != "recent" && sortBy != "helpful" {
		fmt.Println("--sort must be 'recent' or 'helpful'")
		return
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	u, _ := url.Parse(shared.APIURL("/manga/" + mangaID + "/reviews"))
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("sort", sortBy)
	u.RawQuery = q.Encode()

	resp, err := shared.DoAuthReq("GET", u.String(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			fmt.Printf("✗ Error (%s): %s\n", resp.Status, errResp.Error)
		} else {
			fmt.Printf("✗ Error (%s)\n", resp.Status)
		}
		return
	}

	var result struct {
		Items []reviewListItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	if len(result.Items) == 0 {
		fmt.Println("No reviews yet.")
		return
	}

	fmt.Printf("Reviews for %s (%d)\n", mangaID, len(result.Items))
	for _, item := range result.Items {
		timestamp := time.Unix(item.Timestamp, 0).UTC().Format("2006-01-02 15:04:05 UTC")
		fmt.Printf("- %s | %d/10 | helpful: %d | user: %s\n", timestamp, item.Rating, item.Helpful, item.UserID)
		if strings.TrimSpace(item.Text) != "" {
			fmt.Printf("  %s\n", item.Text)
		}
	}
}
