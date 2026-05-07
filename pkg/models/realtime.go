package models

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

type PrivateMessage struct {
	SenderID          string `json:"sender_id"`
	SenderUsername    string `json:"sender_username"`
	RecipientID       string `json:"recipient_id"`
	RecipientUsername string `json:"recipient_username"`
	Message           string `json:"message"`
	Timestamp         int64  `json:"timestamp"`
}

type SendPMRequest struct {
	RecipientUsername string `json:"recipient_username" binding:"required"`
	Message           string `json:"message" binding:"required"`
}

type ConflictResolution struct {
	Strategy   string `json:"strategy"`
	Timestamp  int64  `json:"timestamp"`
	DeviceID   string `json:"device_id"`
	Resolution string `json:"resolution"`
}
