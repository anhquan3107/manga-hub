package manga

import (
	"context"

	"mangahub/internal/cache"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type Service struct {
	store *database.Store
	cache *cache.Client
}

func NewService(store *database.Store) *Service {
	return &Service{store: store}
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

// cache helpers have been moved to cache.go
