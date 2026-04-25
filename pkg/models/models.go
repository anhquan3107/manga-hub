package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
}

type CreateMangaRequest struct {
	ID            string   `json:"id" binding:"required"`
	Title         string   `json:"title" binding:"required"`
	Author        string   `json:"author" binding:"required"`
	Genres        []string `json:"genres" binding:"required,min=1"`
	Status        string   `json:"status" binding:"required"`
	TotalChapters int      `json:"total_chapters" binding:"min=0"`
	Description   string   `json:"description" binding:"required"`
	CoverURL      string   `json:"cover_url"`
}

type UpdateMangaRequest struct {
	Title         string   `json:"title" binding:"required"`
	Author        string   `json:"author" binding:"required"`
	Genres        []string `json:"genres" binding:"required,min=1"`
	Status        string   `json:"status" binding:"required"`
	TotalChapters int      `json:"total_chapters" binding:"min=0"`
	Description   string   `json:"description" binding:"required"`
	CoverURL      string   `json:"cover_url"`
}

type MangaQuery struct {
	Query  string
	Genre  string
	Status string
	Limit  int
}

type LibraryEntry struct {
	UserID         string    `json:"user_id"`
	MangaID        string    `json:"manga_id" binding:"required"`
	Title          string    `json:"title,omitempty"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status" binding:"required"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type AddLibraryRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status" binding:"required"`
}

type UpdateProgressRequest struct {
	MangaID        string `json:"manga_id" binding:"required"`
	CurrentChapter int    `json:"current_chapter" binding:"required,min=0"`
	Status         string `json:"status"`
}

type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

type Notification struct {
	Type      string `json:"type"`
	ClientID  string `json:"client_id,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type ChatMessage struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
