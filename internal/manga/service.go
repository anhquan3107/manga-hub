package manga

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
	mangaCacheTTL        = 2 * time.Minute
	mangaCacheVersionKey = "cache:v1:manga:version"
	mangaCacheNamespace  = "cache:v1:manga"
)

func NewService(store *database.Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetCache(client *cache.Client) {
	s.cache = client
}

func (s *Service) List(ctx context.Context, query models.MangaQuery) ([]models.Manga, error) {
	if s.cache != nil {
		var items []models.Manga
		if ok, err := s.cache.GetJSON(ctx, s.listKey(ctx, query), &items); err == nil && ok {
			return items, nil
		}
	}

	items, err := s.store.ListManga(ctx, query)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, s.listKey(ctx, query), items, mangaCacheTTL)
	}

	return items, nil
}

func (s *Service) GetByID(ctx context.Context, mangaID string) (models.Manga, error) {
	if s.cache != nil {
		var item models.Manga
		if ok, err := s.cache.GetJSON(ctx, s.detailKey(ctx, mangaID), &item); err == nil && ok {
			return item, nil
		}
	}

	item, err := s.store.GetMangaByID(ctx, mangaID)
	if err != nil {
		return models.Manga{}, err
	}

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, s.detailKey(ctx, mangaID), item, mangaCacheTTL)
	}

	return item, nil
}

func (s *Service) Create(ctx context.Context, req models.CreateMangaRequest) (models.Manga, error) {
	item, err := s.store.CreateManga(ctx, models.Manga(req))
	if err != nil {
		return models.Manga{}, err
	}
	_ = s.invalidate(ctx)
	return item, nil
}

func (s *Service) Update(ctx context.Context, mangaID string, req models.UpdateMangaRequest) (models.Manga, error) {
	item, err := s.store.UpdateMangaByID(ctx, mangaID, models.Manga{
		ID:            mangaID,
		Title:         req.Title,
		Author:        req.Author,
		Genres:        req.Genres,
		Status:        req.Status,
		Year:          req.Year,
		Rating:        req.Rating,
		Popularity:    req.Popularity,
		TotalChapters: req.TotalChapters,
		Description:   req.Description,
		CoverURL:      req.CoverURL,
	})
	if err != nil {
		return models.Manga{}, err
	}
	_ = s.invalidate(ctx)
	return item, nil
}

func (s *Service) Delete(ctx context.Context, mangaID string) error {
	if err := s.store.DeleteMangaByID(ctx, mangaID); err != nil {
		return err
	}
	_ = s.invalidate(ctx)
	return nil
}

func (s *Service) invalidate(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Incr(ctx, mangaCacheVersionKey)
	return err
}

func (s *Service) version(ctx context.Context) int64 {
	if s.cache == nil {
		return 0
	}
	version, ok, err := s.cache.GetInt64(ctx, mangaCacheVersionKey)
	if err != nil || !ok {
		return 0
	}
	return version
}

func (s *Service) detailKey(ctx context.Context, mangaID string) string {
	return fmt.Sprintf("%s:detail:v%d:%s", mangaCacheNamespace, s.version(ctx), strings.TrimSpace(mangaID))
}

func (s *Service) listKey(ctx context.Context, query models.MangaQuery) string {
	material := fmt.Sprintf("q=%s|genres=%s|status=%s|year=%d-%d|rating=%.2f|sort=%s|limit=%d",
		strings.TrimSpace(query.Query),
		strings.Join(query.Filters.Genres, ","),
		strings.TrimSpace(query.Filters.Status),
		query.Filters.YearRange[0],
		query.Filters.YearRange[1],
		query.Filters.Rating,
		strings.TrimSpace(query.Filters.SortBy),
		query.Limit,
	)
	hash := sha1.Sum([]byte(material))
	return fmt.Sprintf("%s:list:v%d:%s", mangaCacheNamespace, s.version(ctx), hex.EncodeToString(hash[:]))
}
