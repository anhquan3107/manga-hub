package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

func getAuthClient() *http.Client {
	return &http.Client{}
}

func doAuthReq(method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+loadToken())
	req.Header.Set("Content-Type", "application/json")
	return getAuthClient().Do(req)
}

func HandleLibrary(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub library <add|list> [flags]")
		return
	}
	subCmd := args[0]
	flags := flag.NewFlagSet("library "+subCmd, flag.ExitOnError)

	switch subCmd {
	case "add":
		var mangaID, status string
		flags.StringVar(&mangaID, "manga-id", "", "ID of manga to add")
		flags.StringVar(&status, "status", "reading", "Status (reading/completed/plan-to-read)")
		flags.Parse(args[1:])

		if mangaID == "" {
			fmt.Println("--manga-id required")
			return
		}

		data, _ := json.Marshal(map[string]string{
			"manga_id": mangaID,
			"status":   status,
		})

		resp, err := doAuthReq("POST", "http://localhost:8080/users/library", data)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	case "list":
		resp, err := doAuthReq("GET", "http://localhost:8080/users/library", nil)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}

func HandleProgress(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub progress <update> [flags]")
		return
	}
	subCmd := args[0]
	flags := flag.NewFlagSet("progress "+subCmd, flag.ExitOnError)

	switch subCmd {
	case "update":
		var mangaID string
		var chapter int
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
		fmt.Printf("Status: %s\n", resp.Status)
		printRespBody(resp.Body)

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}
