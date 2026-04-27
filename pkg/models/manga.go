package models

type Manga struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
}

type CreateMangaRequest struct {
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
