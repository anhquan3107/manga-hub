package handler

import "mangahub/pkg/models"

type errorResponse struct {
	Error string `json:"error"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

type mangaListResponse struct {
	Items []models.Manga `json:"items"`
}

type libraryResponse struct {
	Items        []models.LibraryEntry `json:"items"`
	ReadingLists any                   `json:"reading_lists"`
}

type roomsUsersResponse struct {
	Rooms      any `json:"rooms"`
	TotalUsers int `json:"total_users"`
}

type roomUsersResponse struct {
	Users any `json:"users"`
	Count int `json:"count"`
}

type roomHistoryResponse struct {
	Room     string               `json:"room"`
	Limit    int                  `json:"limit"`
	Messages []models.ChatMessage `json:"messages"`
}

type progressHistoryResponse struct {
	Items []models.ProgressHistoryEntry `json:"items"`
}

type progressUpdateResponse struct {
	MangaID         string `json:"manga_id"`
	Title           string `json:"title"`
	PreviousChapter int    `json:"previous_chapter"`
	CurrentChapter  int    `json:"current_chapter"`
	PreviousVolume  int    `json:"previous_volume"`
	CurrentVolume   int    `json:"current_volume"`
	TotalChapters   int    `json:"total_chapters"`
	Notes           string `json:"notes"`
	Status          string `json:"status"`
}

// Keep Swagger schema-only types referenced so static analysis doesn't flag them as unused.
var (
	_ = errorResponse{}
	_ = messageResponse{}
	_ = healthResponse{}
	_ = authResponse{}
	_ = mangaListResponse{}
	_ = libraryResponse{}
	_ = roomsUsersResponse{}
	_ = roomUsersResponse{}
	_ = roomHistoryResponse{}
	_ = progressHistoryResponse{}
	_ = progressUpdateResponse{}
)
