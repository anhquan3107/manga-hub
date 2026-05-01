package manga

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

func handleSearch(args []string) {
	flags := flag.NewFlagSet("manga search", flag.ExitOnError)
	var query, genre, status string
	var limit, page int
	flags.StringVar(&query, "query", "", "Search query for manga")
	flags.StringVar(&genre, "genre", "", "Filter by genre")
	flags.StringVar(&status, "status", "", "Filter by status (ongoing, completed)")
	flags.IntVar(&limit, "limit", 20, "Number of results per page")
	flags.IntVar(&page, "page", 1, "Page number for pagination")

	parseArgs := args
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		query = parseArgs[0]
		parseArgs = parseArgs[1:]
	}

	flags.Parse(parseArgs)

	if strings.TrimSpace(query) == "" {
		fmt.Println("Usage: mangahub manga search <query> [--genre <genre>] [--status <status>] [--limit <n>]")
		return
	}

	fmt.Printf("Searching for \"%s\"...\n", query)

	u, _ := url.Parse("http://localhost:8080/manga")
	q := u.Query()
	if query != "" {
		q.Set("q", query)
	}
	if genre != "" {
		q.Set("genre", genre)
	}
	if status != "" {
		q.Set("status", status)
	}
	q.Set("limit", fmt.Sprintf("%d", limit))
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("✗ Failed to fetch manga: %s\n", resp.Status)
		shared.PrintRespBody(resp.Body)
		return
	}

	var result struct {
		Items []MangaItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	if len(result.Items) == 0 {
		fmt.Println("No manga found matching your search criteria.")
		fmt.Println("Suggestions:")
		fmt.Println("- Check spelling and try again")
		fmt.Println("- Use broader search terms")
		fmt.Println("- Browse by genre: mangahub manga list --genre action")
		return
	}

	fmt.Printf("Found %d results:\n", len(result.Items))
	printMangaTable(result.Items, 0, limit, false)
}

func handleList(args []string) {
	flags := flag.NewFlagSet("manga list", flag.ExitOnError)
	var genre, status string
	var limit, page int
	flags.StringVar(&genre, "genre", "", "Filter by genre")
	flags.StringVar(&status, "status", "", "Filter by status (ongoing, completed)")
	flags.IntVar(&limit, "limit", 20, "Number of results per page")
	flags.IntVar(&page, "page", 1, "Page number for pagination")

	flags.Parse(args)

	u, _ := url.Parse("http://localhost:8080/manga")
	q := u.Query()
	if genre != "" {
		q.Set("genre", genre)
	}
	if status != "" {
		q.Set("status", status)
	}
	q.Set("limit", fmt.Sprintf("%d", limit))
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("✗ Failed to fetch manga: %s\n", resp.Status)
		shared.PrintRespBody(resp.Body)
		return
	}

	var result struct {
		Items []MangaItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error parsing response:", err)
		return
	}

	if len(result.Items) == 0 {
		fmt.Println("No manga found")
		return
	}

	printMangaTable(result.Items, page, limit, true)
}
