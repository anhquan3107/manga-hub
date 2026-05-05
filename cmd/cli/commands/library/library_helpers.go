package library

import (
	"sort"
	"strings"

	shared "mangahub/cmd/cli/commands/shared"
)

type libraryListItem struct {
	MangaID        string `json:"manga_id"`
	Title          string `json:"title"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updated_at"`
	Rating         int    `json:"rating"`
	StartedAt      string `json:"started_at"`
}

func normalizeLibraryStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "completed"
	case "on-hold", "on hold", "on_hold", "onhold":
		return "on-hold"
	case "dropped":
		return "dropped"
	case "plan-to-read", "plan to read", "plantoread", "planned":
		return "plan_to_read"
	default:
		return "reading"
	}
}

func isAllowedLibraryStatus(status string) bool {
	switch normalizeLibraryStatus(status) {
	case "reading", "completed", "plan_to_read", "on-hold", "dropped":
		return true
	default:
		return false
	}
}

func sortLibraryEntries(entries []libraryListItem, sortBy, order string) {
	if sortBy == "" {
		return
	}
	desc := strings.EqualFold(order, "desc")
	sort.SliceStable(entries, func(i, j int) bool {
		switch sortBy {
		case "title":
			left := strings.ToLower(entries[i].Title)
			right := strings.ToLower(entries[j].Title)
			if left == right {
				return entries[i].MangaID < entries[j].MangaID
			}
			if desc {
				return left > right
			}
			return left < right
		case "last-updated":
			if entries[i].UpdatedAt == entries[j].UpdatedAt {
				return entries[i].MangaID < entries[j].MangaID
			}
			if desc {
				return entries[i].UpdatedAt > entries[j].UpdatedAt
			}
			return entries[i].UpdatedAt < entries[j].UpdatedAt
		default:
			return entries[i].MangaID < entries[j].MangaID
		}
	})
}

func truncate(s string, length int) string { return shared.Truncate(s, length) }
