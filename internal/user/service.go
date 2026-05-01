package user

import (
	"context"
	"errors"

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
		Rating:         req.Rating,
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

func (s *Service) RemoveFromLibrary(ctx context.Context, userID, mangaID string) error {
	return s.store.DeleteLibraryEntry(ctx, userID, mangaID)
}

func (s *Service) UpdateLibrary(ctx context.Context, userID, mangaID string, req models.UpdateLibraryRequest) (models.LibraryEntry, error) {
	library, err := s.store.GetUserLibrary(ctx, userID)
	if err != nil {
		return models.LibraryEntry{}, err
	}

	for _, entry := range library {
		if entry.MangaID != mangaID {
			continue
		}

		updated := entry
		if req.Status != "" {
			updated.Status = req.Status
		}
		if req.Rating > 0 {
			updated.Rating = req.Rating
		}

		return s.store.UpsertLibraryEntry(ctx, userID, updated)
	}

	return models.LibraryEntry{}, errors.New("library entry not found")
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	return s.store.GetUserByID(ctx, userID)
}
