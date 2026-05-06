package review

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

func reviewMine(args []string) {
	var mangaID string
	flags := flag.NewFlagSet("review mine", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}

	resp, err := shared.DoAuthReq("GET", shared.APIURL("/manga/"+mangaID+"/reviews/me"), nil)
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

	var review reviewListItem
	if err := json.NewDecoder(resp.Body).Decode(&review); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	timestamp := time.Unix(review.Timestamp, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	fmt.Printf("Your review for %s\n", mangaID)
	fmt.Printf("- %s | %d/10 | helpful: %d\n", timestamp, review.Rating, review.Helpful)
	if strings.TrimSpace(review.Text) != "" {
		fmt.Printf("  %s\n", review.Text)
	}
}
