package manga

import (
	"context"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Service struct {
	store *database.Store
}

func NewService(store *database.Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, query models.MangaQuery) ([]models.Manga, error) {
	return s.store.ListManga(ctx, query)
}

func (s *Service) GetByID(ctx context.Context, mangaID string) (models.Manga, error) {
	return s.store.GetMangaByID(ctx, mangaID)
}
