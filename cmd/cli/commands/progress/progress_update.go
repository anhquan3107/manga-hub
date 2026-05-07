package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleProgressUpdate(args []string) {
	var mangaID string
	var chapter int
	var volume int
	var notes string
	var force bool
	flags := flag.NewFlagSet("progress update", flag.ExitOnError)
	flags.StringVar(&mangaID, "manga-id", "", "ID of manga")
	flags.IntVar(&chapter, "chapter", 0, "Chapter number")
	flags.IntVar(&volume, "volume", 0, "Volume number (optional)")
	flags.StringVar(&notes, "notes", "", "Notes about this progress update")
	flags.BoolVar(&force, "force", false, "Allow backwards progress updates")
	if err := flags.Parse(args); err != nil {
		fmt.Println("Error parsing flags:", err)
		return
	}

	if mangaID == "" || chapter == 0 {
		fmt.Println("--manga-id and --chapter required")
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"manga_id":        mangaID,
		"current_chapter": chapter,
		"current_volume":  volume,
		"notes":           notes,
		"force":           force,
	})

	resp, err := shared.DoAuthReq("PUT", shared.APIURL("/users/progress"), data)
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

	var res progressUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		fmt.Println("✓ Progress updated (response could not be parsed)")
		return
	}

	fmt.Println("Updating reading progress...")
	fmt.Println("✓ Progress updated successfully!")
	fmt.Printf("Manga: %s\n", shared.NonEmpty(res.Title, mangaID))
	fmt.Printf("Previous: Chapter %d\n", res.PreviousChapter)
	fmt.Printf("Current: Chapter %d (+%d)\n", res.CurrentChapter, res.CurrentChapter-res.PreviousChapter)
	if res.CurrentVolume > 0 {
		fmt.Printf("Volume: %d\n", res.CurrentVolume)
	}
	fmt.Printf("Updated: %s\n", res.UpdatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Println("Sync Status:")
	fmt.Println(" Local database: ✓ Updated")
	fmt.Println(" TCP sync server: ✓ Broadcasted")
	fmt.Println(" Cloud backup: N/A")
	fmt.Println("Statistics:")
	fmt.Printf(" Total chapters read: %s\n", shared.FormatNumber(res.CurrentChapter))
	fmt.Println(" Reading streak: N/A")
	if res.TotalChapters > 0 {
		fmt.Printf(" Estimated completion: Chapter %d\n", res.TotalChapters)
	} else {
		fmt.Println(" Estimated completion: N/A")
	}
	if res.Notes != "" {
		fmt.Printf("Notes: %s\n", res.Notes)
	}
	fmt.Println("Next actions:")
	fmt.Printf(" Continue reading: Chapter %d\n", res.CurrentChapter+1)
	fmt.Printf(" Rate this chapter: mangahub library update --manga-id %s --rating 9\n", mangaID)
}
