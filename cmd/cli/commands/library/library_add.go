package library

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func libraryAdd(args []string) {
	var mangaID, status string
	var rating int
	flags := flag.NewFlagSet("library add", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga to add")
	flags.StringVar(&status, "status", "reading", "Status (reading/completed/plan-to-read/on-hold/dropped)")
	flags.IntVar(&rating, "rating", 0, "Optional rating (1-10)")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}
	if !isAllowedLibraryStatus(status) {
		fmt.Println("✗ Invalid status. Use one of: reading, completed, plan-to-read, on-hold, dropped")
		return
	}

	payload := map[string]interface{}{"manga_id": mangaID, "status": status}
	if rating > 0 {
		payload["rating"] = rating
	}
	data, _ := json.Marshal(payload)

	resp, err := shared.DoAuthReq("POST", shared.APIURL("/users/library"), data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		errorText := strings.ToLower(strings.TrimSpace(errResp.Error))
		if resp.StatusCode == http.StatusNotFound || strings.Contains(errorText, "no rows") || strings.Contains(errorText, "foreign key") || strings.Contains(errorText, "manga not found") {
			fmt.Printf("✗ Manga '%s' not found. Please search for manga first:\n", mangaID)
			fmt.Println("  mangahub manga search <keyword>")
			return
		}
		if errResp.Error != "" {
			fmt.Printf("✗ Error (%s): %s\n", resp.Status, errResp.Error)
		} else {
			fmt.Printf("✗ Error (%s)\n", resp.Status)
		}
		return
	}

	fmt.Printf("✓ Added '%s' to library with status '%s'\n", mangaID, status)
}
