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
	Status         string    `json:"status" binding:"required,oneof=reading completed plan-to-read on-hold dropped"`
	UpdatedAt      time.Time `json:"updated_at"`
	Rating         int       `json:"rating,omitempty"`
	StartedAt      time.Time `json:"started_at,omitempty"`
}

type AddLibraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status" binding:"required,oneof=reading completed plan-to-read on-hold dropped"`
	Rating         int    `json:"rating,omitempty"`
}

type UpdateProgressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"required,min=0"`
	Status         string `json:"status" binding:"omitempty,oneof=reading completed plan-to-read on-hold dropped"`
}
