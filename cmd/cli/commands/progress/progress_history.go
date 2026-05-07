package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleProgressHistory(args []string) {
	var mangaID string
	flags := flag.NewFlagSet("progress history", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga (optional)")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	u, _ := url.Parse(shared.APIURL("/users/progress/history"))
	if mangaID != "" {
		q := u.Query()
		q.Set("manga_id", mangaID)
		u.RawQuery = q.Encode()
	}

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
		fmt.Printf("✗ Error: %s\n", errResp.Error)
		return
	}

	var res progressHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		fmt.Println("No history found")
		return
	}

	if len(res.Items) == 0 {
		fmt.Println("No progress history available")
		return
	}

	fmt.Println("Progress History:")
	for _, item := range res.Items {
		fmt.Printf("- %s | %s: %d -> %d (vol %d -> %d)\n",
			item.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
			item.MangaID,
			item.PreviousChapter,
			item.CurrentChapter,
			item.PreviousVolume,
			item.CurrentVolume,
		)
		if item.Notes != "" {
			fmt.Printf("  Notes: %s\n", item.Notes)
		}
	}
}
