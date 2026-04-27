package models

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type LibraryEntry struct {
	UserID         int64     `json:"user_id"`
	MangaID        int64     `json:"manga_id" binding:"required"`
	Title          string    `json:"title,omitempty"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status" binding:"required"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AddLibraryRequest struct {
	MangaID        int64  `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status" binding:"required"`
}

type UpdateProgressRequest struct {
	MangaID        int64  `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"required,min=0"`
	Status         string `json:"status"`
}
