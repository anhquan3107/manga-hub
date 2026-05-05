package review

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func reviewAdd(args []string) {
	var mangaID string
	var rating int
	var text string
	flags := flag.NewFlagSet("review add", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga to review")
	flags.IntVar(&rating, "rating", 0, "Rating (1-10)")
	flags.StringVar(&text, "text", "", "Review text")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}
	if rating < 1 || rating > 10 {
		fmt.Println("--rating must be between 1 and 10")
		return
	}
	if strings.TrimSpace(text) == "" {
		fmt.Println("--text required")
		return
	}

	payload := map[string]any{
		"rating": rating,
		"text":   text,
	}
	data, _ := json.Marshal(payload)

	resp, err := shared.DoAuthReq("POST", shared.APIURL("/manga/"+mangaID+"/reviews"), data)
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

	fmt.Printf("✓ Review saved for '%s' (%d/10)\n", mangaID, rating)
}
