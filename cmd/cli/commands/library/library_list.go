package library

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func libraryList(args []string) {
	var statusFilter string
	var sortBy string
	var order string
	var verbose bool
	flags := flag.NewFlagSet("library list", flag.ExitOnError)
	flags.StringVar(&statusFilter, "status", "", "Filter by status")
	flags.StringVar(&sortBy, "sort-by", "", "Sort by (title,last-updated)")
	flags.StringVar(&order, "order", "", "Order (asc,desc)")
	flags.BoolVar(&verbose, "verbose", false, "Verbose output with descriptions")
	flags.Parse(args)

	u := "http://localhost:8080/users/library"
	if statusFilter != "" || sortBy != "" || order != "" {
		params := "?"
		if statusFilter != "" {
			params += "status=" + statusFilter + "&"
		}
		if sortBy != "" {
			params += "sort_by=" + sortBy + "&"
		}
		if order != "" {
			params += "order=" + order + "&"
		}
		u += strings.TrimRight(params, "&")
	}

	resp, err := shared.DoAuthReq("GET", u, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Status: %s\n", resp.Status)
		shared.PrintRespBody(resp.Body)
		return
	}

	var result struct {
		Items        []libraryListItem            `json:"items"`
		ReadingLists map[string][]libraryListItem `json:"reading_lists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	entries := result.Items
	if statusFilter != "" {
		wantedStatus := normalizeLibraryStatus(statusFilter)
		filtered := make([]libraryListItem, 0, len(entries))
		for _, entry := range entries {
			if normalizeLibraryStatus(entry.Status) == wantedStatus {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	total := len(entries)
	if total == 0 {
		if statusFilter != "" {
			fmt.Printf("No library entries found for status '%s'.\n", statusFilter)
		} else {
			fmt.Println("Your library is empty.")
		}
		fmt.Println("Get started by searching and adding manga:")
		fmt.Println("  mangahub manga search \"your favorite series\"")
		fmt.Println("  mangahub library add --manga-id <id> --status reading")
		return
	}

	fmt.Printf("Your Manga Library (%d entries)\n", total)
	orderSections := []string{"reading", "completed", "plan_to_read", "on-hold", "dropped"}
	for _, section := range orderSections {
		list := make([]libraryListItem, 0)
		for _, entry := range entries {
			if normalizeLibraryStatus(entry.Status) == section {
				list = append(list, entry)
			}
		}
		if len(list) == 0 {
			continue
		}
		sortLibraryEntries(list, sortBy, order)
		title := strings.Title(strings.ReplaceAll(section, "_", " "))
		fmt.Printf("%s (%d):\n", title, len(list))
		fmt.Println("┌──────────────────┬────────────────────────┬─────────┬────────────┬──────────┐")
		fmt.Println("│ ID               │ Title                  │ Chapter │ Rating     │ Started  │")
		fmt.Println("├──────────────────┼────────────────────────┼─────────┼────────────┼──────────┤")
		for _, e := range list {
			id := truncate(e.MangaID, 16)
			title := truncate(e.Title, 22)
			chapter := fmt.Sprintf("%d/??", e.CurrentChapter)
			rating := "Unrated"
			if e.Rating > 0 {
				rating = fmt.Sprintf("%d/10", e.Rating)
			}
			started := "-"
			if e.StartedAt != "" {
				started = e.StartedAt
			}
			fmt.Printf("│ %-16s │ %-22s │ %-7s │ %-10s │ %-8s │\n", id, title, chapter, rating, started)
		}
		fmt.Println("└──────────────────┴────────────────────────┴─────────┴────────────┴──────────┘")
		fmt.Println()
	}
	_ = verbose
}
