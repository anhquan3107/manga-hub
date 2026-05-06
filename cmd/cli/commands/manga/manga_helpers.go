package manga

import (
	"fmt"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type MangaItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	Year          int      `json:"year"`
	Rating        float64  `json:"rating"`
	Popularity    int      `json:"popularity"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
}

func printMangaTable(items []MangaItem, page, limit int, showFooter bool) {
	if page > 0 {
		fmt.Printf("✓ Found %d manga\n\n", len(items))
	}
	fmt.Println("┌────────────────────┬────────────────────────┬──────────────┬────────────┬──────────┐")
	fmt.Println("│ ID                 │ Title                  │ Author       │ Status     │ Chapters │")
	fmt.Println("├────────────────────┼────────────────────────┼──────────────┼────────────┼──────────┤")
	for _, item := range items {
		id := shared.Truncate(item.ID, 18)
		title := shared.Truncate(item.Title, 22)
		author := shared.Truncate(item.Author, 12)
		status := shared.Truncate(cases.Title(language.English).String(item.Status), 10)
		fmt.Printf("│ %-18s │ %-22s │ %-12s │ %-10s │ %8d │\n", id, title, author, status, item.TotalChapters)
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
	centeredTitle := shared.CenterText(titleLine, width)
	fmt.Printf("\n┌%s┐\n", horizontal)
	fmt.Printf("│%s│\n", centeredTitle)
	fmt.Printf("└%s┘\n", horizontal)

	fmt.Println("Basic Information:")
	fmt.Printf(" ID: %s\n", shared.NonEmpty(manga.ID, "-"))
	fmt.Printf(" Title: %s\n", shared.NonEmpty(manga.Title, "-"))
	fmt.Printf(" Author: %s\n", shared.NonEmpty(manga.Author, "-"))
	fmt.Printf(" Artist: %s\n", shared.NonEmpty(manga.Author, "-"))
	if len(manga.Genres) > 0 {
		fmt.Printf(" Genres: %s\n", strings.Join(manga.Genres, ", "))
	} else {
		fmt.Println(" Genres: -")
	}
	fmt.Printf(" Status: %s\n", shared.NonEmpty(cases.Title(language.English).String(manga.Status), "-"))
	if manga.Year > 0 {
		fmt.Printf(" Year: %d\n", manga.Year)
	} else {
		fmt.Println(" Year: -")
	}
	if manga.Rating > 0 {
		fmt.Printf(" Rating: %.1f\n", manga.Rating)
	} else {
		fmt.Println(" Rating: -")
	}
	if manga.Popularity > 0 {
		fmt.Printf(" Popularity: %s\n", shared.FormatNumber(manga.Popularity))
	} else {
		fmt.Println(" Popularity: -")
	}

	fmt.Println("Progress:")
	if manga.TotalChapters > 0 {
		fmt.Printf(" Total Chapters: %s+\n", shared.FormatNumber(manga.TotalChapters))
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
		for _, line := range shared.WrapText(desc, width) {
			fmt.Printf(" %s\n", line)
		}
	}

	fmt.Println("External Links:")
	fmt.Printf(" MyAnimeList: https://placeholder.local/mal/%s\n", shared.NonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" MangaDx: https://placeholder.local/mangadx/%s\n", shared.NonEmpty(manga.ID, "manga-id"))
	if manga.CoverURL != "" {
		fmt.Printf(" Cover: %s\n", manga.CoverURL)
	}

	fmt.Println("Actions:")
	fmt.Printf(" Update Progress: mangahub progress update --manga-id %s --chapter <number>\n", shared.NonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" Rate/Review: mangahub review add --manga-id %s --rating <1-10> --text \"...\"\n", shared.NonEmpty(manga.ID, "manga-id"))
	fmt.Printf(" Remove: mangahub library remove --manga-id %s\n", shared.NonEmpty(manga.ID, "manga-id"))
}
