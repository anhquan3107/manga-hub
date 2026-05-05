package models

type Review struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Rating    int    `json:"rating"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
	Helpful   int    `json:"helpful"`
}

type CreateReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=10"`
	Text   string `json:"text" binding:"required"`
}
