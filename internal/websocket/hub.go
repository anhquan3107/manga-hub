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
	RoomID   string
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*gorillaws.Conn]ClientConnection
	Rooms      map[string][]ClientConnection
	Register   chan ClientConnection
	Unregister chan *gorillaws.Conn
	Broadcast  chan RoomMessage
	Private    chan PrivateDelivery
}

type RoomMessage struct {
	RoomID  string
	Message models.ChatMessage
}

type PrivateDelivery struct {
	RecipientID string
	Message     models.PrivateMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*gorillaws.Conn]ClientConnection),
		Rooms:      make(map[string][]ClientConnection),
		Register:   make(chan ClientConnection),
		Unregister: make(chan *gorillaws.Conn),
		Broadcast:  make(chan RoomMessage, 32),
		Private:    make(chan PrivateDelivery, 32),
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
			h.Rooms[client.RoomID] = append(h.Rooms[client.RoomID], client)
			h.mu.Unlock()
			h.Broadcast <- RoomMessage{
				RoomID: client.RoomID,
				Message: models.ChatMessage{
					UserID:    client.UserID,
					Username:  client.Username,
					Message:   "joined the chat",
					Timestamp: time.Now().Unix(),
				},
			}
		case conn := <-h.Unregister:
			h.mu.Lock()
			client, ok := h.clients[conn]
			if ok {
				delete(h.clients, conn)
				for i, c := range h.Rooms[client.RoomID] {
					if c.Conn == conn {
						h.Rooms[client.RoomID] = append(h.Rooms[client.RoomID][:i], h.Rooms[client.RoomID][i+1:]...)
						break
					}
				}
			}
			h.mu.Unlock()
			_ = conn.Close()
			if ok {
				h.Broadcast <- RoomMessage{
					RoomID: client.RoomID,
					Message: models.ChatMessage{
						UserID:    client.UserID,
						Username:  client.Username,
						Message:   "left the chat",
						Timestamp: time.Now().Unix(),
					},
				}
			}
		case msg := <-h.Broadcast:
			h.mu.RLock()
			failed := make([]*gorillaws.Conn, 0)
			for _, client := range h.Rooms[msg.RoomID] {
				if err := client.Conn.WriteJSON(msg.Message); err != nil {
					log.Printf("websocket write error: %v", err)
					failed = append(failed, client.Conn)
				}
			}
			h.mu.RUnlock()
			for _, conn := range failed {
				h.removeClient(conn)
			}
		case pm := <-h.Private:
			h.mu.RLock()
			failed := make([]*gorillaws.Conn, 0)
			for _, client := range h.clients {
				if client.UserID != pm.RecipientID {
					continue
				}
				payload := map[string]any{
					"type":      "pm",
					"user_id":   pm.Message.SenderID,
					"username":  pm.Message.SenderUsername,
					"message":   "[PM] " + pm.Message.Message,
					"timestamp": pm.Message.Timestamp,
				}
				if err := client.Conn.WriteJSON(payload); err != nil {
					log.Printf("websocket private write error: %v", err)
					failed = append(failed, client.Conn)
				}
			}
			h.mu.RUnlock()
			for _, conn := range failed {
				h.removeClient(conn)
			}
		}
	}
}

func (h *Hub) removeClient(conn *gorillaws.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	_ = conn.Close()
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		_ = conn.Close()
		delete(h.clients, conn)
	}
}

func (h *Hub) GetRoomUsers(roomID string) []ClientConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]ClientConnection, len(h.Rooms[roomID]))
	copy(users, h.Rooms[roomID])
	return users
}

func (h *Hub) GetAllRoomUsers() map[string][]ClientConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rooms := make(map[string][]ClientConnection, len(h.Rooms))
	for roomID, users := range h.Rooms {
		copied := make([]ClientConnection, len(users))
		copy(copied, users)
		rooms[roomID] = copied
	}

	return rooms
}
