package review

import (
	"context"
	"fmt"
	"time"

	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Service struct {
	store *database.Store
}

func NewService(store *database.Store) *Service {
	return &Service{store: store}
}

func (s *Service) UpsertReview(ctx context.Context, userID, mangaID string, req models.CreateReviewRequest) (models.Review, error) {
	if _, err := s.store.GetMangaByID(ctx, mangaID); err != nil {
		return models.Review{}, err
	}

	review := models.Review{
		UserID:    userID,
		MangaID:   mangaID,
		Rating:    req.Rating,
		Text:      req.Text,
		Timestamp: time.Now().Unix(),
	}

	return s.store.UpsertReview(ctx, review)
}

func (s *Service) GetReview(ctx context.Context, userID, mangaID string) (models.Review, error) {
	return s.store.GetReview(ctx, userID, mangaID)
}

func (s *Service) ListReviews(ctx context.Context, mangaID string, limit int, sortBy string) ([]models.Review, error) {
	if _, err := s.store.GetMangaByID(ctx, mangaID); err != nil {
		return nil, err
	}

	return s.store.ListReviewsByManga(ctx, mangaID, limit, sortBy)
}

func (s *Service) IncrementHelpful(ctx context.Context, userID, mangaID string) (models.Review, error) {
	review, err := s.store.IncrementReviewHelpful(ctx, userID, mangaID)
	if err != nil {
		return models.Review{}, err
	}
	return review, nil
}

func ValidateReviewRating(rating int) error {
	if rating < 1 || rating > 10 {
		return fmt.Errorf("rating must be between 1 and 10")
	}
	return nil
}
