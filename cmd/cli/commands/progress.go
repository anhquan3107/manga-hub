package commands

import (
	"encoding/json"
	"flag"
	"fmt"
)

func progressHandler(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub progress <update> [flags]")
		return
	}
	subCmd := args[0]
	switch subCmd {
	case "update":
		var mangaID string
		var chapter int
		flags := flag.NewFlagSet("progress update", flag.ExitOnError)
		flags.StringVar(&mangaID, "manga-id", "", "ID of manga")
		flags.IntVar(&chapter, "chapter", 0, "Chapter number")
		flags.Parse(args[1:])

		if mangaID == "" || chapter == 0 {
			fmt.Println("--manga-id and --chapter required")
			return
		}

		data, _ := json.Marshal(map[string]interface{}{
			"manga_id": mangaID,
			"chapter":  chapter,
		})

		resp, err := doAuthReq("PUT", "http://localhost:8080/users/progress", data)
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

		fmt.Printf("✓ Updated progress for '%s' to chapter %d\n", mangaID, chapter)

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
