package commands

import (
	"net"
	"time"

	shared "mangahub/cmd/cli/commands/shared"
	"mangahub/pkg/models"
)

type tcpMessage struct {
    Type      string `json:"type"`
    RequestID string `json:"request_id,omitempty"`
    UserID    string `json:"user_id,omitempty"`
    MangaID   string `json:"manga_id,omitempty"`
    Chapter   int    `json:"chapter,omitempty"`
}

type tcpResponse struct {
    Type        string                 `json:"type"`
    RequestID   string                 `json:"request_id,omitempty"`
    Message     string                 `json:"message,omitempty"`
    Error       string                 `json:"error,omitempty"`
    Progress    *models.ProgressUpdate `json:"progress,omitempty"`
    Username    string                 `json:"username,omitempty"`
    UserID      string                 `json:"user_id,omitempty"`
    SessionID   string                 `json:"session_id,omitempty"`
    ConnectedAt int64                  `json:"connected_at,omitempty"`
    Devices     int                    `json:"devices,omitempty"`
    Timestamp   int64                  `json:"timestamp"`
}

type Session struct {
    SessionID   string `json:"session_id"`
    ConnectedAt int64  `json:"connected_at"`
}

var (
    tcpConn       net.Conn
    tcpAddr       = shared.TCPAddr()
    connectedAt   time.Time
    lastHeartbeat time.Time
    messagesSent  int
    messagesRecv  int
)
