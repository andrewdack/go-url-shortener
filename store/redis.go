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

// delCachedUrl deletes shorturl mapping from the redis cache
func (s *StorageService) delCachedUrl(ctx context.Context, shortUrl string) error {
	// make sure to delete shortUrl key, as well as clicksKey
	if err := s.redisClient.Del(ctx, shortUrl, clicksKey(shortUrl)).Err(); err != nil {
		return fmt.Errorf("failed evicting cache for %s: %v", shortUrl, err)
	}
	return nil
}

// setCachedUrlClickCounter caches the click counter for shortUrl.
// Note that it DOES not increment the click counter itself, it is solely for caching the current count, which comes from the database.
func (s *StorageService) setCachedUrlClickCounter(ctx context.Context, shortUrl string, count int) error {
	if err := s.redisClient.Set(ctx, clicksKey(shortUrl), count, 0).Err(); err != nil {
		return fmt.Errorf("failed caching click counter for %s: %w", shortUrl, err)
	}
	return nil
}

// getCachedUrlClickCounter returns the cached click count for shortUrl.
// Returns redis.Nil (wrapped) if there's no cache entry — callers should
// treat any error here as a cache miss and fall back to Postgres.
func (s *StorageService) getCachedUrlClickCounter(ctx context.Context, shortUrl string) (int64, error) {
	count, err := s.redisClient.Get(ctx, clicksKey(shortUrl)).Int64()
	if err != nil {
		return 0, fmt.Errorf("failed retrieving cached click counter for %s: %w", shortUrl, err)
	}
	return count, nil
}
