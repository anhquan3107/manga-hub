package review

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"mangahub/internal/cache"
	"strings"
)

const (
	reviewCacheTTL        = 2 * 60 // seconds
	reviewCacheVersionKey = "cache:v1:review:version"
	reviewCacheNamespace  = "cache:v1:review"
)

func (s *Service) SetCache(client *cache.Client) {
	s.cache = client
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
