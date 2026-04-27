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

func (s *Service) Create(ctx context.Context, req models.CreateMangaRequest) (models.Manga, error) {
	return s.store.CreateManga(ctx, models.Manga{
		ID:            req.ID,
		Title:         req.Title,
		Author:        req.Author,
		Genres:        req.Genres,
		Status:        req.Status,
		TotalChapters: req.TotalChapters,
		Description:   req.Description,
		CoverURL:      req.CoverURL,
	})
}

func (s *Service) Update(ctx context.Context, mangaID string, req models.UpdateMangaRequest) (models.Manga, error) {
	return s.store.UpdateMangaByID(ctx, mangaID, models.Manga{
		ID:            mangaID,
		Title:         req.Title,
		Author:        req.Author,
		Genres:        req.Genres,
		Status:        req.Status,
		TotalChapters: req.TotalChapters,
		Description:   req.Description,
		CoverURL:      req.CoverURL,
	})
}

func (s *Service) Delete(ctx context.Context, mangaID string) error {
	return s.store.DeleteMangaByID(ctx, mangaID)
}
