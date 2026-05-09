package manga

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mangahub/internal/cache"
	"mangahub/pkg/models"
)

const (
	mangaCacheTTL        = 2 * time.Minute
	mangaCacheVersionKey = "cache:v1:manga:version"
	mangaCacheNamespace  = "cache:v1:manga"
)

// SetCache sets the redis cache client on the service (dependency injection).
func (s *Service) SetCache(client *cache.Client) {
	s.cache = client
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
