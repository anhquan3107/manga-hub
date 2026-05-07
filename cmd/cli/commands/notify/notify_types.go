package commands

import (
	"net"
)

type udpClientMessage struct {
    Type    string `json:"type"`
    Client  string `json:"client,omitempty"`
    MangaID string `json:"manga_id,omitempty"`
}

type udpServerMessage struct {
    Type    string `json:"type"`
    Message string `json:"message,omitempty"`
}

var (
    notifyUDPAddr *net.UDPAddr
    registeredClientID string
)
