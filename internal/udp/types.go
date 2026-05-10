package udp

import (
	"net"
	"time"

	"mangahub/pkg/models"
)

type registeredClient struct {
	ID        string
	Addr      *net.UDPAddr
	LastSeen  time.Time
	Connected time.Time
}

type clientMessage struct {
	Type string `json:"type"`
	// Accept both "client_id" (server tests) and "client" (CLI)
	ClientID  string `json:"client_id,omitempty"`
	Client    string `json:"client,omitempty"`
	MangaID   string `json:"manga_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type serverMessage struct {
	Type      string               `json:"type"`
	ClientID  string               `json:"client_id,omitempty"`
	Message   string               `json:"message,omitempty"`
	Error     string               `json:"error,omitempty"`
	Payload   *models.Notification `json:"payload,omitempty"`
	Timestamp int64                `json:"timestamp"`
}
