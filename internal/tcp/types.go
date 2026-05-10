package tcp

import (
	"net"
	"sync"

	"mangahub/pkg/models"
)

type client struct {
	id     string
	conn   net.Conn
	userID string
	mu     sync.Mutex
}

func (c *client) setUserID(userID string) {
	c.mu.Lock()
	c.userID = userID
	c.mu.Unlock()
}

func (c *client) getUserID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.userID
}

type clientMessage struct {
	Type      string                 `json:"type"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	MangaID   string                 `json:"manga_id,omitempty"`
	Chapter   int                    `json:"chapter,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Strategy  string                 `json:"strategy,omitempty"`
	Progress  *models.ProgressUpdate `json:"progress,omitempty"`
}

type serverMessage struct {
	Type        string                     `json:"type"`
	RequestID   string                     `json:"request_id,omitempty"`
	Message     string                     `json:"message,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Progress    *models.ProgressUpdate     `json:"progress,omitempty"`
	Conflict    *models.ConflictResolution `json:"conflict,omitempty"`
	SessionID   string                     `json:"session_id,omitempty"`
	Username    string                     `json:"username,omitempty"`
	UserID      string                     `json:"user_id,omitempty"`
	Devices     int                        `json:"devices,omitempty"`
	ConnectedAt int64                      `json:"connected_at,omitempty"`
	Timestamp   int64                      `json:"timestamp"`
}
