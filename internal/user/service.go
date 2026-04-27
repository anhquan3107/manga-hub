package user

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

func (s *Service) AddToLibrary(ctx context.Context, userID string, req models.AddLibraryRequest) (models.LibraryEntry, error) {
	if _, err := s.store.GetMangaByID(ctx, req.MangaID); err != nil {
		return models.LibraryEntry{}, err
	}

	return s.store.UpsertLibraryEntry(ctx, userID, models.LibraryEntry{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         req.Status,
	})
}

func (s *Service) UpdateProgress(ctx context.Context, userID string, req models.UpdateProgressRequest) (models.LibraryEntry, error) {
	status := req.Status
	if status == "" {
		status = "reading"
	}

	return s.store.UpsertLibraryEntry(ctx, userID, models.LibraryEntry{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         status,
	})
}

func (s *Service) GetLibrary(ctx context.Context, userID string) ([]models.LibraryEntry, error) {
	return s.store.GetUserLibrary(ctx, userID)
}
