package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

// define a struct wrapper around raw Redis client and Postgres connection pool
type StorageService struct {
	redisClient *redis.Client
	pgPool      *pgxpool.Pool
}

// Note that in real world the cache duration shouldn't have an expiration time
// but rather an LRU policy where values that are retrieved less often are
// purged automatically from the cache and stored back in RDBMS whenever cache is full
const CacheDuration = 6 * time.Hour

// NewStore initializes and returns a new instance of StorageService with Redis and Postgres connections set up.
// It returns an error if any of the initializations fail.
func NewStore(ctx context.Context) (*StorageService, error) {
	// Initialize Redis Cache
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("Error init Redis: %v", err)
	}
	fmt.Printf("\nRedis started successfully: pong message = {%s}", pong)

	// Initialize Postgres
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		databaseUrl = "postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable"
	}
	pgPool, err := pgxpool.New(ctx, databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("Error init postgres: %v", err)
	}

	if err := pgPool.Ping(ctx); err != nil {
		panic(fmt.Sprintf("Error connecting to Postgres: %v", err))
	}
	fmt.Println("\nPostgres started successfully")
	
	return &StorageService{
		redisClient: redisClient,
		pgPool: pgPool,
	}, nil
}

// Close closes the Redis client and Postgres connection pool associated with the storage service.
// It returns an error if closing the Redis client fails. The Postgres connection pool is closed regardless of errors.
func (s *StorageService) Close() error {
	var closeErr error

	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			closeErr = fmt.Errorf("failed closing Redis client: %w", err)
		}
	}
	if s.pgPool != nil {
		s.pgPool.Close()
	}
	return closeErr
}

// SaveUrlMapping saves a mapping between a short URL and its original long URL in the storage service.
// Returns an error if the operation fails.
func (s *StorageService) SaveUrlMapping(ctx context.Context, shortUrl string, originalUrl string, userId string) error {
	if err := s.insertUrl(ctx, shortUrl, originalUrl, userId); err != nil {
		return err // includes ErrAlreadyExists
	}

	if err := s.setCachedUrl(ctx, shortUrl, originalUrl); err != nil {
		log.Printf("failed warming cache for %s: %v", shortUrl, err)
	}

	return nil
}

// RetrieveInitialUrl retrieves the original long URL associated with a given short URL from the storage service.
// Returns the long URL and an error if the operation fails.
// Will be used when a user accesses the short URL to retrieve the original long URL for redirection.
func (s *StorageService) RetrieveInitialUrl(ctx context.Context, shortUrl string) (string, error) {
	cached, err := s.getCachedUrl(ctx, shortUrl)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, redis.Nil) {
		log.Printf("cache read failed for %s, falling back to postgres: %v", shortUrl, err)
	}

	longUrl, _, err := s.getUrlByShortCode(ctx, shortUrl)
	if err != nil {
		return "", err
	}

	if err := s.setCachedUrl(ctx, shortUrl, longUrl); err != nil {
		log.Printf("failed warming cache for %s: %v", shortUrl, err)
	}
	return longUrl, nil
}

// DeleteUrlMapping deletes a shortUrl key mapping from the storage service
// Returns the deleted Url and errors, if any.
func (s *StorageService) DeleteUrlMapping(ctx context.Context, shortUrl string, userId string) (deletedUrl string, err error) {
	_, storedUserId, err := s.getUrlByShortCode(ctx, shortUrl)
	if err != nil {
		return "", err
	}
	if storedUserId != userId {
		return "", ErrUserIdMismatch
	}

	if err := s.deleteUrl(ctx, shortUrl); err != nil {
		return "", err
	}

	if err := s.redisClient.Del(ctx, shortUrl, clicksKey(shortUrl)).Err(); err != nil {
		log.Printf("failed evicting cache for %s: %v", shortUrl, err)
	}
	return shortUrl, nil
}

// UpdateUrlMapping updates the long URL associated with a given short URL in the storage service.
// Returns the new long URL and errors, if any.
func (s *StorageService) UpdateUrlMapping(ctx context.Context, shortUrl string, newLongUrl string, userId string) (updatedUrl string, err error) {
	oldLongUrl, storedUserId, err := s.getUrlByShortCode(ctx, shortUrl)
	if err != nil {
		return "", err
	}
	if storedUserId != userId {
		return oldLongUrl, ErrUserIdMismatch
	}

	if err := s.updateUrl(ctx, shortUrl, newLongUrl); err != nil {
		return oldLongUrl, err
	}

	if err := s.setCachedUrl(ctx, shortUrl, newLongUrl); err != nil {
		log.Printf("failed refreshing cache for %s: %v", shortUrl, err)
	}
	return newLongUrl, nil
}

// IncrementRateLimitCounter increments the request counter for key and, if this
// is the first request in a new window, sets the counter to expire after window.
// Returns the counter's value after incrementing.
func (s *StorageService) IncrementRateLimitCounter(ctx context.Context, key string, window time.Duration) (int64, error) {
	count, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed incrementing rate limit counter %s: %w", key, err)
	}

	if count == 1 {
		if err := s.redisClient.Expire(ctx, key, window).Err(); err != nil {
			return 0, fmt.Errorf("failed setting expiry on rate limit counter %s: %w", key, err)
		}
	}
	return count, nil
}

// IncrementClickCount increments the click count for a given short URL key in the storage service.
// Returns the updated click count and errors, if any.
func (s *StorageService) IncrementClickCount(ctx context.Context, shortUrl string) (int64, error) {
	key := clicksKey(shortUrl)
	count, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed incrementing click for %s: %w", shortUrl, err)
	}

	if count == 1 {
		if err := s.redisClient.Expire(ctx, key, CacheDuration).Err(); err != nil {
			return 0, fmt.Errorf("failed setting ex for %s: %w", shortUrl, err)
		}
	}
	return count, nil
}

// RetrieveClickCount returns the number of successful redirects for a short URL.
func (s *StorageService) RetrieveClickCount(ctx context.Context, shortUrl string) (int64, error) {
	if _, err := s.RetrieveInitialUrl(ctx, shortUrl); err != nil {
		return 0, err
	}

	count, err := s.redisClient.Get(ctx, clicksKey(shortUrl)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed retrieving click count for %s: %w", shortUrl, err)
	}
	return count, nil
}
