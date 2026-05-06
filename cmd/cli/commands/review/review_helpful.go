package review

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	shared "mangahub/cmd/cli/commands/shared"
)

func reviewHelpful(args []string) {
	var mangaID string
	var userID string
	flags := flag.NewFlagSet("review helpful", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga")
	flags.StringVar(&userID, "user-id", "", "User ID of review author")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}
	if userID == "" {
		fmt.Println("--user-id required")
		return
	}

	resp, err := shared.DoAuthReq("POST", shared.APIURL("/manga/"+mangaID+"/reviews/"+userID+"/helpful"), nil)
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

	fmt.Printf("✓ Marked review helpful for user '%s' on '%s'\n", userID, mangaID)
}
