package models

type Manga struct {
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

type CreateMangaRequest struct {
	ID            string   `json:"id" binding:"required"`
	Title         string   `json:"title" binding:"required"`
	Author        string   `json:"author" binding:"required"`
	Genres        []string `json:"genres" binding:"required,min=1"`
	Status        string   `json:"status" binding:"required"`
	Year          int      `json:"year" binding:"omitempty,min=0"`
	Rating        float64  `json:"rating" binding:"omitempty,min=0,max=10"`
	Popularity    int      `json:"popularity" binding:"omitempty,min=0"`
	TotalChapters int      `json:"total_chapters" binding:"min=0"`
	Description   string   `json:"description" binding:"required"`
	CoverURL      string   `json:"cover_url"`
}

type UpdateMangaRequest struct {
	Title         string   `json:"title" binding:"required"`
	Author        string   `json:"author" binding:"required"`
	Genres        []string `json:"genres" binding:"required,min=1"`
	Status        string   `json:"status" binding:"required"`
	Year          int      `json:"year" binding:"omitempty,min=0"`
	Rating        float64  `json:"rating" binding:"omitempty,min=0,max=10"`
	Popularity    int      `json:"popularity" binding:"omitempty,min=0"`
	TotalChapters int      `json:"total_chapters" binding:"min=0"`
	Description   string   `json:"description" binding:"required"`
	CoverURL      string   `json:"cover_url"`
}

type SearchFilters struct {
	Genres    []string
	Status    string
	YearRange [2]int
	Rating    float64
	SortBy    string
}

type MangaQuery struct {
	Query   string
	Filters SearchFilters
	Limit   int
}
