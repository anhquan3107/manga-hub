package chat

import (
	"context"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Service struct {
	store *database.Store
}

func NewService(store *database.Store) *Service {
	return &Service{
		store: store,
	}
}

// SaveMessage saves a chat message to the database
func (s *Service) SaveMessage(ctx context.Context, msg models.ChatMessage, roomID string) error {
	return s.store.InsertChatMessage(ctx, msg, roomID)
}

// GetRoomHistory retrieves recent chat messages for a room
func (s *Service) GetRoomHistory(ctx context.Context, roomID string, limit int) ([]models.ChatMessage, error) {
	return s.store.ListChatMessages(ctx, roomID, limit)
}
