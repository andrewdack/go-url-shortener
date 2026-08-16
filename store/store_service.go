package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// define a struct wrapper around raw Redis client and Postgres connection pool
type StorageService struct {
	redisClient *redis.Client
	pgPool      *pgxpool.Pool
}

// NewStore initializes and returns a new instance of StorageService with Redis and Postgres connections set up.
// It returns an error if any of the initializations fail.
func NewStore(ctx context.Context) (*StorageService, error) {
	// Initialize Redis Cache
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_ADDR")
	}
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	redisOptions := &redis.Options{Addr: redisURL}
	if strings.Contains(redisURL, "://") {
		var err error
		redisOptions, err = redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("error parsing Redis URL: %v", err)
		}
	}

	redisClient := redis.NewClient(redisOptions)

	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		_ = redisClient.Close()
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
		_ = redisClient.Close()
		return nil, fmt.Errorf("Error init postgres: %v", err)
	}

	if err := pgPool.Ping(ctx); err != nil {
		pgPool.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("error connecting to Postgres: %w", err)
	}
	fmt.Println("\nPostgres started successfully")

	return &StorageService{
		redisClient: redisClient,
		pgPool:      pgPool,
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
	if err := s.dbInsertUrl(ctx, shortUrl, originalUrl, userId); err != nil {
		if !errors.Is(err, ErrAlreadyExists) {
			return err
		}

		// Short codes are deterministic for a given URL and user. Treat a
		// repeat submission as success, but keep true hash collisions visible.
		storedUrl, storedUserID, lookupErr := s.dbGetUrl(ctx, shortUrl)
		if lookupErr != nil {
			return lookupErr
		}
		if storedUrl != originalUrl || storedUserID != userId {
			return ErrAlreadyExists
		}
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

	longUrl, _, err := s.dbGetUrl(ctx, shortUrl)
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
	_, storedUserId, err := s.dbGetUrl(ctx, shortUrl)
	if err != nil {
		return "", err
	}
	if storedUserId != userId {
		return "", ErrUserIdMismatch
	}

	if err := s.dbDeleteUrl(ctx, shortUrl); err != nil {
		return "", err
	}

	if err := s.delCachedUrl(ctx, shortUrl); err != nil {
		log.Printf("error deleting cached url: %v", err)
	}
	return shortUrl, nil
}

// UpdateUrlMapping updates the long URL associated with a given short URL in the storage service.
// Returns the new long URL and errors, if any.
func (s *StorageService) UpdateUrlMapping(ctx context.Context, shortUrl string, newLongUrl string, userId string) (updatedUrl string, err error) {
	oldLongUrl, storedUserId, err := s.dbGetUrl(ctx, shortUrl)
	if err != nil {
		return "", err
	}
	if storedUserId != userId {
		return oldLongUrl, ErrUserIdMismatch
	}

	if err := s.dbUpdateUrl(ctx, shortUrl, newLongUrl); err != nil {
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
	// only postgres does the incrementing
	count, err := s.dbIncrementClickCount(ctx, shortUrl)

	if errors.Is(err, pgx.ErrNoRows) {
		return -1, ErrNotFound
	}
	if err != nil {
		return -1, fmt.Errorf("error incrementing click counter for %v", shortUrl)
	}

	// refresh the click cache for Redis to copy postgres
	if err := s.setCachedUrlClickCounter(ctx, shortUrl, int(count)); err != nil {
		log.Printf("error: %v", err)
	}

	return count, nil
}

// RetrieveClickCount returns the number of successful redirects for a short URL.
func (s *StorageService) RetrieveClickCount(ctx context.Context, shortUrl string) (int64, error) {
	cached, err := s.getCachedUrlClickCounter(ctx, shortUrl)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, redis.Nil) {
		log.Printf("click count cache read failed for %s, falling back to postgres: %v", shortUrl, err)
	}

	count, err := s.dbGetClickCount(ctx, shortUrl)
	if err != nil {
		return 0, err
	}

	if err := s.setCachedUrlClickCounter(ctx, shortUrl, int(count)); err != nil {
		log.Printf("failed warming click count cache for %s: %v", shortUrl, err)
	}
	return count, nil
}
