package manga

import (
	"encoding/json"
	"fmt"
	"net/http"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleInfo(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub manga info <manga-id>")
		return
	}
	id := args[0]
	resp, err := http.Get(shared.APIURL("/manga/" + id))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("✗ Manga not found: %s\n", resp.Status)
		return
	}
	var manga MangaItem
	if err := json.NewDecoder(resp.Body).Decode(&manga); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}
	printMangaDetail(manga)
}
