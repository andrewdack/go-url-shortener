package store

import (
	"context"
	"fmt"
)

// getCachedUrl returns the long URL cached for shortUrl.
// Returns redis.Nil (wrapped) if there's no cache entry — callers should
// treat any error here as a cache miss and fall back to Postgres.
func (s *StorageService) getCachedUrl(ctx context.Context, shortUrl string) (string, error) {
	result, err := s.redisClient.Get(ctx, shortUrl).Result()
	if err != nil {
		return "", fmt.Errorf("failed retrieving cached url %s: %w", shortUrl, err)
	}
	return result, nil
}

// setCachedUrl caches longUrl for shortUrl.
func (s *StorageService) setCachedUrl(ctx context.Context, shortUrl string, longUrl string) error {
	if err := s.redisClient.Set(ctx, shortUrl, longUrl, CacheDuration).Err(); err != nil {
		return fmt.Errorf("failed caching url %s: %w", shortUrl, err)
	}
	return nil
}
