package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type LibraryEntry struct {
	UserID         string    `json:"user_id"`
	MangaID        string    `json:"manga_id" binding:"required"`
	Title          string    `json:"title,omitempty"`
	CurrentChapter int       `json:"current_chapter"`
	CurrentVolume  int       `json:"current_volume,omitempty"`
	Status         string    `json:"status" binding:"required,oneof=reading completed plan-to-read on-hold dropped"`
	UpdatedAt      time.Time `json:"updated_at"`
	Rating         int       `json:"rating,omitempty"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

type AddLibraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	CurrentVolume  int    `json:"current_volume,omitempty"`
	Status         string `json:"status" binding:"required,oneof=reading completed plan-to-read on-hold dropped"`
	Rating         int    `json:"rating,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type UpdateProgressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"required,min=0"`
	CurrentVolume  int    `json:"current_volume" binding:"omitempty,min=0"`
	Status         string `json:"status" binding:"omitempty,oneof=reading completed plan-to-read on-hold dropped"`
	Notes          string `json:"notes"`
	Force          bool   `json:"force"`
}

type UpdateLibraryRequest struct {
	Status string `json:"status" binding:"omitempty,oneof=reading completed plan-to-read on-hold dropped"`
	Rating int    `json:"rating" binding:"omitempty,min=1,max=10"`
}

type ProgressHistoryEntry struct {
	ID             int64     `json:"id"`
	UserID         string    `json:"user_id"`
	MangaID        string    `json:"manga_id"`
	PreviousChapter int      `json:"previous_chapter"`
	CurrentChapter int       `json:"current_chapter"`
	PreviousVolume int       `json:"previous_volume"`
	CurrentVolume  int       `json:"current_volume"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProgressUpdateResult struct {
	Entry           LibraryEntry `json:"entry"`
	PreviousChapter int          `json:"previous_chapter"`
	PreviousVolume  int          `json:"previous_volume"`
	TotalChapters   int          `json:"total_chapters"`
	MangaTitle      string       `json:"manga_title"`
}
