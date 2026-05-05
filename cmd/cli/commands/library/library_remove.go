package library

import (
	"encoding/json"
	"flag"
	"fmt"

	shared "mangahub/cmd/cli/commands/shared"
)

func libraryRemove(args []string) {
	var mangaID string
	flags := flag.NewFlagSet("library remove", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga to remove")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}

	resp, err := shared.DoAuthReq("DELETE", shared.APIURL("/users/library/"+mangaID), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("✗ Error: %s\n", errResp.Error)
		return
	}

	fmt.Printf("✓ Removed '%s' from library\n", mangaID)
}
