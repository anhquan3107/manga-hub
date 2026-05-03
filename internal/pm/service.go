package pm

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

// SendMessage sends a private message from sender to recipient
func (s *Service) SendMessage(ctx context.Context, pm models.PrivateMessage) error {
	return s.store.SendPrivateMessage(ctx, pm)
}

// GetReceivedMessages retrieves private messages sent to a user
func (s *Service) GetReceivedMessages(ctx context.Context, userID string, limit int) ([]models.PrivateMessage, error) {
	return s.store.ListPrivateMessages(ctx, userID, limit)
}
