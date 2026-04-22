package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"mangahub/pkg/models"
)

type ClientConnection struct {
	Conn     *gorillaws.Conn
	UserID   string
	Username string
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*gorillaws.Conn]ClientConnection
	Register   chan ClientConnection
	Unregister chan *gorillaws.Conn
	Broadcast  chan models.ChatMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*gorillaws.Conn]ClientConnection),
		Register:   make(chan ClientConnection),
		Unregister: make(chan *gorillaws.Conn),
		Broadcast:  make(chan models.ChatMessage, 32),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.Conn] = client
			h.mu.Unlock()
			h.Broadcast <- models.ChatMessage{
				UserID:    client.UserID,
				Username:  client.Username,
				Message:   "joined the chat",
				Timestamp: time.Now().Unix(),
			}
		case conn := <-h.Unregister:
			h.mu.Lock()
			client, ok := h.clients[conn]
			if ok {
				delete(h.clients, conn)
			}
			h.mu.Unlock()
			_ = conn.Close()
			if ok {
				h.Broadcast <- models.ChatMessage{
					UserID:    client.UserID,
					Username:  client.Username,
					Message:   "left the chat",
					Timestamp: time.Now().Unix(),
				}
			}
		case message := <-h.Broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteJSON(message); err != nil {
					log.Printf("websocket write error: %v", err)
					_ = conn.Close()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		_ = conn.Close()
		delete(h.clients, conn)
	}
}
