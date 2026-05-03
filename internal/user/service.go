package user

import (
	"context"
	"errors"
	"fmt"

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
		CurrentVolume:  req.CurrentVolume,
		Status:         req.Status,
		Rating:         req.Rating,
		Notes:          req.Notes,
	})
}

func (s *Service) UpdateProgress(ctx context.Context, userID string, req models.UpdateProgressRequest) (models.ProgressUpdateResult, error) {
	status := req.Status
	if status == "" {
		status = "reading"
	}

	manga, err := s.store.GetMangaByID(ctx, req.MangaID)
	if err != nil {
		return models.ProgressUpdateResult{}, fmt.Errorf("manga not found")
	}

	entry, err := s.store.GetLibraryEntry(ctx, userID, req.MangaID)
	if err != nil {
		return models.ProgressUpdateResult{}, fmt.Errorf("manga '%s' not found in your library", req.MangaID)
	}

	if manga.TotalChapters > 0 && req.CurrentChapter > manga.TotalChapters {
		return models.ProgressUpdateResult{}, fmt.Errorf("chapter %d exceeds manga's total chapters (%d)", req.CurrentChapter, manga.TotalChapters)
	}

	if req.CurrentChapter < entry.CurrentChapter && !req.Force {
		return models.ProgressUpdateResult{}, fmt.Errorf("chapter %d is behind your current progress (chapter %d)", req.CurrentChapter, entry.CurrentChapter)
	}

	newVolume := entry.CurrentVolume
	if req.CurrentVolume > 0 {
		if req.CurrentVolume < entry.CurrentVolume && !req.Force {
			return models.ProgressUpdateResult{}, fmt.Errorf("volume %d is behind your current progress (volume %d)", req.CurrentVolume, entry.CurrentVolume)
		}
		newVolume = req.CurrentVolume
	}

	newNotes := entry.Notes
	if req.Notes != "" {
		newNotes = req.Notes
	}

	updated, err := s.store.UpsertLibraryEntry(ctx, userID, models.LibraryEntry{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		CurrentVolume:  newVolume,
		Status:         status,
		Notes:          newNotes,
	})
	if err != nil {
		return models.ProgressUpdateResult{}, err
	}

	_ = s.store.InsertProgressHistory(ctx, models.ProgressHistoryEntry{
		UserID:         userID,
		MangaID:        req.MangaID,
		PreviousChapter: entry.CurrentChapter,
		CurrentChapter: req.CurrentChapter,
		PreviousVolume: entry.CurrentVolume,
		CurrentVolume:  newVolume,
		Notes:          req.Notes,
	})

	return models.ProgressUpdateResult{
		Entry:           updated,
		PreviousChapter: entry.CurrentChapter,
		PreviousVolume:  entry.CurrentVolume,
		TotalChapters:   manga.TotalChapters,
		MangaTitle:      manga.Title,
	}, nil
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

func (s *Service) GetProgressHistory(ctx context.Context, userID, mangaID string) ([]models.ProgressHistoryEntry, error) {
	return s.store.GetProgressHistory(ctx, userID, mangaID)
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	return s.store.GetUserByID(ctx, userID)
}
