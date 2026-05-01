package library

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func libraryUpdate(args []string) {
	var mangaID, status string
	var rating int
	flags := flag.NewFlagSet("library update", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga to update")
	flags.StringVar(&status, "status", "", "New status (reading/completed/plan-to-read/on-hold/dropped)")
	flags.IntVar(&rating, "rating", 0, "New rating (1-10, use 0 to skip)")
	flags.Parse(args)

	if mangaID == "" {
		fmt.Println("--manga-id required")
		return
	}
	if status == "" && rating == 0 {
		fmt.Println("At least one of --status or --rating must be provided")
		return
	}
	if status != "" && !isAllowedLibraryStatus(status) {
		fmt.Println("✗ Invalid status. Use one of: reading, completed, plan-to-read, on-hold, dropped")
		return
	}

	payload := map[string]interface{}{}
	if status != "" {
		payload["status"] = status
	}
	if rating > 0 {
		payload["rating"] = rating
	}
	data, _ := json.Marshal(payload)

	resp, err := shared.DoAuthReq("PUT", "http://localhost:8080/users/library/"+mangaID, data)
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
		errorText := strings.ToLower(strings.TrimSpace(errResp.Error))
		if resp.StatusCode == http.StatusNotFound || strings.Contains(errorText, "not found") {
			fmt.Printf("✗ Manga '%s' not found in library\n", mangaID)
			return
		}
		if errResp.Error != "" {
			fmt.Printf("✗ Error (%s): %s\n", resp.Status, errResp.Error)
		} else {
			fmt.Printf("✗ Error (%s)\n", resp.Status)
		}
		return
	}

	var updated libraryListItem
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	changes := make([]string, 0, 2)
	if status != "" {
		changes = append(changes, "status "+updated.Status)
	}
	if rating > 0 {
		changes = append(changes, fmt.Sprintf("rating %d/10", updated.Rating))
	}

	fmt.Printf("✓ Updated '%s' (%s)\n", mangaID, strings.Join(changes, ", "))
}
