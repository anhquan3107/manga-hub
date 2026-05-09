package user

import (
    "context"
    "fmt"
    "strings"
    "mangahub/internal/cache"
)

const (
    userCacheTTL        = 5 * 60
    userLookupCacheTTL  = 5 * 60
    userLibraryCacheTTL = 2 * 60
    userHistoryCacheTTL = 2 * 60
    userCacheVersionKey = "cache:v1:user:version"
    userCacheNamespace  = "cache:v1:user"
)

func (s *Service) SetCache(client *cache.Client) {
    s.cache = client
}

func (s *Service) invalidate(ctx context.Context) error {
    if s.cache == nil {
        return nil
    }
    _, err := s.cache.Incr(ctx, userCacheVersionKey)
    return err
}

func (s *Service) version(ctx context.Context) int64 {
    if s.cache == nil {
        return 0
    }
    version, ok, err := s.cache.GetInt64(ctx, userCacheVersionKey)
    if err != nil || !ok {
        return 0
    }
    return version
}

func (s *Service) userKey(ctx context.Context, userID string) string {
    return fmt.Sprintf("%s:detail:v%d:%s", userCacheNamespace, s.version(ctx), strings.TrimSpace(userID))
}

func (s *Service) usernameKey(username string) string {
    return fmt.Sprintf("%s:username:%s", userCacheNamespace, strings.ToLower(strings.TrimSpace(username)))
}

func (s *Service) libraryKey(ctx context.Context, userID string) string {
    return fmt.Sprintf("%s:library:v%d:%s", userCacheNamespace, s.version(ctx), strings.TrimSpace(userID))
}

func (s *Service) historyKey(ctx context.Context, userID, mangaID string) string {
    return fmt.Sprintf("%s:history:v%d:%s:%s", userCacheNamespace, s.version(ctx), strings.TrimSpace(userID), strings.TrimSpace(mangaID))
}
