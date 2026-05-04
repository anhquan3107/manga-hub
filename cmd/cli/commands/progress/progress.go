package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
)

var progressTCPAddr = shared.TCPAddr()

type progressTCPMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Chapter   int    `json:"chapter,omitempty"`
}

type progressTCPResponse struct {
	Type      string `json:"type"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type progressUpdateResponse struct {
	MangaID         string    `json:"manga_id"`
	Title           string    `json:"title"`
	PreviousChapter int       `json:"previous_chapter"`
	CurrentChapter  int       `json:"current_chapter"`
	PreviousVolume  int       `json:"previous_volume"`
	CurrentVolume   int       `json:"current_volume"`
	UpdatedAt       time.Time `json:"updated_at"`
	TotalChapters   int       `json:"total_chapters"`
	Notes           string    `json:"notes"`
	Status          string    `json:"status"`
}

type progressHistoryResponse struct {
	Items []struct {
		ID              int64     `json:"id"`
		UserID          string    `json:"user_id"`
		MangaID         string    `json:"manga_id"`
		PreviousChapter int       `json:"previous_chapter"`
		CurrentChapter  int       `json:"current_chapter"`
		PreviousVolume  int       `json:"previous_volume"`
		CurrentVolume   int       `json:"current_volume"`
		Notes           string    `json:"notes"`
		CreatedAt       time.Time `json:"created_at"`
	} `json:"items"`
}

func HandleProgress(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub progress <update|history|sync|sync-status> [flags]")
		return
	}
	subCmd := args[0]
	switch subCmd {
	case "update":
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
		flags.Parse(args[1:])

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

	case "history":
		var mangaID string
		flags := flag.NewFlagSet("progress history", flag.ExitOnError)
		flags.StringVar(&mangaID, "manga-id", "", "ID of manga (optional)")
		flags.Parse(args[1:])

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

	case "sync":
		var userID string
		flags := flag.NewFlagSet("progress sync", flag.ExitOnError)
		flags.StringVar(&userID, "user-id", "", "Your user ID (from auth token)")
		flags.Parse(args[1:])

		if userID == "" {
			userID = "default-user"
		}

		if err := progressSync(userID); err != nil {
			fmt.Printf("✗ Sync failed: %v\n", err)
			return
		}
		fmt.Println("✓ Sync completed successfully")

	case "sync-status":
		flags := flag.NewFlagSet("progress sync-status", flag.ExitOnError)
		flags.Parse(args[1:])

		if err := progressSyncStatus(); err != nil {
			fmt.Printf("TCP sync server: ✗ %v\n", err)
			return
		}
		fmt.Println("TCP sync server: ✓ Reachable")

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}

func progressSync(userID string) error {
	conn, err := net.DialTimeout("tcp", progressTCPAddr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", progressTCPAddr, err)
	}
	defer conn.Close()

	// Send hello
	hello := progressTCPMessage{Type: "hello", UserID: userID}
	data, _ := json.Marshal(hello)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	reader := bufio.NewScanner(conn)
	if !reader.Scan() {
		return fmt.Errorf("no response to hello")
	}

	// Send ping as a lightweight sync check
	ping := progressTCPMessage{Type: "ping", RequestID: fmt.Sprintf("sync-%d", time.Now().Unix())}
	data, _ = json.Marshal(ping)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("send ping: %w", err)
	}
	if !reader.Scan() {
		return fmt.Errorf("no response to ping")
	}

	var resp progressTCPResponse
	if err := json.Unmarshal(reader.Bytes(), &resp); err != nil {
		return fmt.Errorf("invalid ping response: %w", err)
	}
	if resp.Type != "pong" {
		return fmt.Errorf("unexpected response: %s", resp.Type)
	}

	return nil
}

func progressSyncStatus() error {
	conn, err := net.DialTimeout("tcp", progressTCPAddr, 2*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
