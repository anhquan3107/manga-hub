package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

type MangaItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
}

func HandleManga(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mangahub manga <search|list|info> [flags]")
		return
	}

	subCmd := args[0]
	flags := flag.NewFlagSet("manga "+subCmd, flag.ExitOnError)
	var query, genre, status string
	var limit, page int
	flags.StringVar(&query, "query", "", "Search query for manga")
	flags.StringVar(&genre, "genre", "", "Filter by genre")
	flags.StringVar(&status, "status", "", "Filter by status (ongoing, completed)")
	flags.IntVar(&limit, "limit", 20, "Number of results per page")
	flags.IntVar(&page, "page", 1, "Page number for pagination")

	switch subCmd {
	case "search", "list":
		parseArgs := args[1:]
		if subCmd == "search" && len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
			query = parseArgs[0]
			parseArgs = parseArgs[1:]
		}

		flags.Parse(parseArgs)

		if subCmd == "search" && strings.TrimSpace(query) == "" {
			fmt.Println("Usage: mangahub manga search <query> [--genre <genre>] [--status <status>] [--limit <n>]")
			return
		}

		if subCmd == "search" {
			fmt.Printf("Searching for \"%s\"...\n", query)
		}

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
			printRespBody(resp.Body)
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
			if subCmd == "search" {
				fmt.Println("No manga found matching your search criteria.")
				fmt.Println("Suggestions:")
				fmt.Println("- Check spelling and try again")
				fmt.Println("- Use broader search terms")
				fmt.Println("- Browse by genre: mangahub manga list --genre action")
				return
			}

			fmt.Println("No manga found")
			return
		}

		if subCmd == "search" {
			fmt.Printf("Found %d results:\n", len(result.Items))
			printMangaTable(result.Items, 0, limit, false)
			return
		}

		printMangaTable(result.Items, page, limit, true)

	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: mangahub manga info <manga-id>")
			return
		}
		id := args[1]
		resp, err := http.Get("http://localhost:8080/manga/" + id)
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

	default:
		fmt.Println("Unknown subcommand:", subCmd)
	}
}

func printMangaTable(items []MangaItem, page, limit int, showFooter bool) {
	if page > 0 {
		fmt.Printf("✓ Found %d manga\n\n", len(items))
	}

	// Table header
	fmt.Println("┌────────────────────┬────────────────────────┬──────────────┬────────────┬──────────┐")
	fmt.Println("│ ID                 │ Title                  │ Author       │ Status     │ Chapters │")
	fmt.Println("├────────────────────┼────────────────────────┼──────────────┼────────────┼──────────┤")

	// Table rows
	for _, item := range items {
		id := truncate(item.ID, 18)
		title := truncate(item.Title, 22)
		author := truncate(item.Author, 12)
		status := truncate(strings.Title(item.Status), 10)
		fmt.Printf("│ %-18s │ %-22s │ %-12s │ %-10s │ %8d │\n",
			id, title, author, status, item.TotalChapters)
	}

	fmt.Println("└────────────────────┴────────────────────────┴──────────────┴────────────┴──────────┘")
	if showFooter {
		if page > 0 {
			fmt.Printf("\nPage: %d | Limit: %d per page\n", page, limit)
		} else {
			fmt.Printf("\nLimit: %d per page\n", limit)
		}
	}
}

func printMangaDetail(manga MangaItem) {
	titleLine := strings.ToUpper(strings.TrimSpace(manga.Title))
	if titleLine == "" {
		titleLine = strings.ToUpper(manga.ID)
	}

	const width = 69
	horizontal := strings.Repeat("─", width)
	centeredTitle := centerText(titleLine, width)
	fmt.Printf("\n┌%s┐\n", horizontal)
	fmt.Printf("│%s│\n", centeredTitle)
	fmt.Printf("└%s┘\n", horizontal)

	fmt.Println("Basic Information:")
	fmt.Printf(" ID: %s\n", nonEmpty(manga.ID, "-"))
	fmt.Printf(" Title: %s\n", nonEmpty(manga.Title, "-"))
	fmt.Printf(" Author: %s\n", nonEmpty(manga.Author, "-"))
	fmt.Printf(" Artist: %s\n", nonEmpty(manga.Author, "-"))
	if len(manga.Genres) > 0 {
		fmt.Printf(" Genres: %s\n", strings.Join(manga.Genres, ", "))
	} else {
		fmt.Println(" Genres: -")
	}
	fmt.Printf(" Status: %s\n", nonEmpty(strings.Title(manga.Status), "-"))
	fmt.Println(" Year: -")

	fmt.Println("Progress:")
	if manga.TotalChapters > 0 {
		fmt.Printf(" Total Chapters: %s+\n", formatNumber(manga.TotalChapters))
	} else {
		fmt.Println(" Total Chapters: -")
	}
	fmt.Println(" Total Volumes: -")
	fmt.Println(" Serialization: -")
	fmt.Println(" Publisher: -")
	fmt.Println("Your Status: -")
	fmt.Println(" Current Chapter: -")
	fmt.Println(" Last Updated: -")
	fmt.Println(" Started Reading: -")
	fmt.Println(" Personal Rating: -")

	fmt.Println("Description:")
	desc := strings.TrimSpace(manga.Description)
	if desc == "" {
		fmt.Println(" -")
	} else {
		for _, line := range wrapText(desc, width) {
			fmt.Printf(" %s\n", line)
		}
	}

	fmt.Println("External Links:")
	fmt.Printf(" MyAnimeList: https://placeholder.local/mal/%s\n", nonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" MangaDx: https://placeholder.local/mangadx/%s\n", nonEmpty(manga.ID, "manga-id"))
	if manga.CoverURL != "" {
		fmt.Printf(" Cover: %s\n", manga.CoverURL)
	}

	fmt.Println("Actions:")
	fmt.Printf(" Update Progress: mangahub progress update --manga-id %s --chapter <number>\n", nonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" Rate/Review: mangahub library update --manga-id %s --rating <1-10>\n", nonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" Remove: mangahub library remove --manga-id %s\n", nonEmpty(manga.ID, "manga-id"))
}

func truncate(s string, length int) string {
	if len(s) > length {
		return s[:length-1] + "…"
	}
	return s
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatNumber(v int) string {
	s := strconv.Itoa(v)
	if len(s) <= 3 {
		return s
	}

	first := len(s) % 3
	if first == 0 {
		first = 3
	}

	parts := []string{s[:first]}
	for i := first; i < len(s); i += 3 {
		parts = append(parts, s[i:i+3])
	}
	return strings.Join(parts, ",")
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	lines := make([]string, 0, len(words)/2)
	current := words[0]

	for _, word := range words[1:] {
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}

	lines = append(lines, current)
	return lines
}

func centerText(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		r = r[:width]
	}
	trimmed := string(r)
	visibleLen := utf8.RuneCountInString(trimmed)
	if visibleLen >= width {
		return trimmed
	}

	pad := width - visibleLen
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + trimmed + strings.Repeat(" ", right)
}
