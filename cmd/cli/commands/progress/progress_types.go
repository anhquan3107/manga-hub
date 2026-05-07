package commands

import (
	"time"
)

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
