package review

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mangahub/internal/cache"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Service struct {
	store *database.Store
	cache *cache.Client
}

const (
	reviewCacheTTL        = 2 * time.Minute
	reviewCacheVersionKey = "cache:v1:review:version"
	reviewCacheNamespace  = "cache:v1:review"
)

func NewService(store *database.Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetCache(client *cache.Client) {
	s.cache = client
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

	item, err := s.store.UpsertReview(ctx, review)
	if err != nil {
		return models.Review{}, err
	}
	_ = s.invalidate(ctx)
	return item, nil
}

func (s *Service) GetReview(ctx context.Context, userID, mangaID string) (models.Review, error) {
	if s.cache != nil {
		var item models.Review
		if ok, err := s.cache.GetJSON(ctx, s.reviewKey(ctx, userID, mangaID), &item); err == nil && ok {
			return item, nil
		}
	}

	item, err := s.store.GetReview(ctx, userID, mangaID)
	if err != nil {
		return models.Review{}, err
	}

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, s.reviewKey(ctx, userID, mangaID), item, reviewCacheTTL)
	}

	return item, nil
}

func (s *Service) ListReviews(ctx context.Context, mangaID string, limit int, sortBy string) ([]models.Review, error) {
	if _, err := s.store.GetMangaByID(ctx, mangaID); err != nil {
		return nil, err
	}

	if s.cache != nil {
		var items []models.Review
		if ok, err := s.cache.GetJSON(ctx, s.listKey(ctx, mangaID, limit, sortBy), &items); err == nil && ok {
			return items, nil
		}
	}

	items, err := s.store.ListReviewsByManga(ctx, mangaID, limit, sortBy)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, s.listKey(ctx, mangaID, limit, sortBy), items, reviewCacheTTL)
	}

	return items, nil
}

func (s *Service) IncrementHelpful(ctx context.Context, userID, mangaID string) (models.Review, error) {
	review, err := s.store.IncrementReviewHelpful(ctx, userID, mangaID)
	if err != nil {
		return models.Review{}, err
	}
	_ = s.invalidate(ctx)
	return review, nil
}

func ValidateReviewRating(rating int) error {
	if rating < 1 || rating > 10 {
		return fmt.Errorf("rating must be between 1 and 10")
	}
	return nil
}

func (s *Service) invalidate(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Incr(ctx, reviewCacheVersionKey)
	return err
}

func (s *Service) version(ctx context.Context) int64 {
	if s.cache == nil {
		return 0
	}
	version, ok, err := s.cache.GetInt64(ctx, reviewCacheVersionKey)
	if err != nil || !ok {
		return 0
	}
	return version
}

func (s *Service) reviewKey(ctx context.Context, userID, mangaID string) string {
	return fmt.Sprintf("%s:mine:v%d:%s:%s", reviewCacheNamespace, s.version(ctx), strings.TrimSpace(userID), strings.TrimSpace(mangaID))
}

func (s *Service) listKey(ctx context.Context, mangaID string, limit int, sortBy string) string {
	material := fmt.Sprintf("manga=%s|limit=%d|sort=%s", strings.TrimSpace(mangaID), limit, strings.TrimSpace(sortBy))
	hash := sha1.Sum([]byte(material))
	return fmt.Sprintf("%s:list:v%d:%s", reviewCacheNamespace, s.version(ctx), hex.EncodeToString(hash[:]))
}
