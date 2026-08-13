package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

// define a struct wrapper around raw Redis client
type StorageService struct {
	redisClient *redis.Client
}

// Top level declarations for the storeService and Redis context
var (
	storeService = &StorageService{}
	ctx          = context.Background()
)

// Note that in real world the cache duration shouldn't have an expiration time
// but rather an LRU policy where values that are retrieved less often are
// purged automatically from the cache and stored back in RDBMS whenever cache is full
const CacheDuration = 6 * time.Hour

var ErrUserIdMismatch = errors.New("user ID does not own short URL")

func ownerKey(shortUrl string) string {
	return fmt.Sprintf("owner:%s", shortUrl)
}

func clicksKey(shortUrl string) string {
	return fmt.Sprintf("clicks:%s", shortUrl)
}

// Initializing the store service and return a store pointer
func InitializeStore() *StorageService {
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
		panic(fmt.Sprintf("Error init Redis: %v", err))
	}
	fmt.Printf("\nRedis started successfully: pong message = {%s}", pong)
	storeService.redisClient = redisClient
	return storeService
}

// SaveUrlMapping saves a mapping between a short URL and its original long URL in the storage service.
// Returns an error if the operation fails.
func SaveUrlMapping(shortUrl string, originalUrl string, userId string) error {
	_, err := storeService.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, shortUrl, originalUrl, CacheDuration)
		pipe.Set(ctx, ownerKey(shortUrl), userId, CacheDuration)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed saving key url - shortUrl: %s - originalUrl: %s: %w", shortUrl, originalUrl, err)
	}
	return nil
}

// RetrieveInitialUrl retrieves the original long URL associated with a given short URL from the storage service.
// Returns the long URL and an error if the operation fails.
// Will be used when a user accesses the short URL to retrieve the original long URL for redirection.
func RetrieveInitialUrl(shortUrl string) (string, error) {
	result, err := storeService.redisClient.Get(ctx, shortUrl).Result()
	if err != nil {
		return "", fmt.Errorf("failed retrieving short url %s: %w", shortUrl, err)
	}
	return result, nil
}

// DeleteUrlMapping deletes a shortUrl key mapping from the storage service
// Returns the deleted Url and errors, if any.
func DeleteUrlMapping(shortUrl string, userId string) (deletedUrl string, err error) {
	_, err = RetrieveInitialUrl(shortUrl)
	if err != nil {
		return "", err
	}

	storedUserId, err := storeService.redisClient.Get(ctx, ownerKey(shortUrl)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrUserIdMismatch
	}
	if err != nil {
		return "", fmt.Errorf("failed retrieving owner for short url %s: %w", shortUrl, err)
	}
	if storedUserId != userId {
		return "", ErrUserIdMismatch
	}

	_, err = storeService.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, shortUrl, ownerKey(shortUrl), clicksKey(shortUrl))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed deleting short url %s: %w", shortUrl, err)
	}
	return shortUrl, err
}

// UpdateUrlMapping updates the long URL associated with a given short URL in the storage service.
// Returns the new long URL and errors, if any.
func UpdateUrlMapping(shortUrl string, newLongUrl string, userId string) (updatedUrl string, err error) {
	oldLongUrl, err := RetrieveInitialUrl(shortUrl)
	if err != nil {
		return "", err
	}

	storedUserId, err := storeService.redisClient.Get(ctx, ownerKey(shortUrl)).Result()
	if errors.Is(err, redis.Nil) {
		return oldLongUrl, ErrUserIdMismatch
	}
	if err != nil {
		return oldLongUrl, fmt.Errorf("failed retrieving owner for short url %s: %w", shortUrl, err)
	}
	if storedUserId != userId {
		return oldLongUrl, ErrUserIdMismatch
	}

	_, err = storeService.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, shortUrl, newLongUrl, CacheDuration)
		pipe.Expire(ctx, ownerKey(shortUrl), CacheDuration)
		return nil
	})

	if err != nil {
		return oldLongUrl, fmt.Errorf("failed updating short url %s: %w", shortUrl, err)
	}
	return newLongUrl, nil
}

// IncrementRateLimitCounter increments the request counter for key and, if this
// is the first request in a new window, sets the counter to expire after window.
// Returns the counter's value after incrementing.
func IncrementRateLimitCounter(key string, window time.Duration) (int64, error) {
	count, err := storeService.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed incrementing rate limit counter %s: %w", key, err)
	}

	if count == 1 {
		if err := storeService.redisClient.Expire(ctx, key, window).Err(); err != nil {
			return 0, fmt.Errorf("failed setting expiry on rate limit counter %s: %w", key, err)
		}
	}
	return count, nil
}

// IncrementClickCount increments the click count for a given short URL key in the storage service.
// Returns the updated click count and errors, if any.
func IncrementClickCount(key string, shortUrl string) (int64, error) {
	count, err := storeService.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed incrementing click for %s: %w", shortUrl, err)
	}

	if count == 1 {
		if err := storeService.redisClient.Expire(ctx, key, CacheDuration).Err(); err != nil {
			return 0, fmt.Errorf("failed setting ex for %s: %w", shortUrl, err)
		}
	}
	return count, nil
}

// RetrieveClickCount returns the number of successful redirects for a short URL.
func RetrieveClickCount(shortUrl string) (int64, error) {
	if _, err := RetrieveInitialUrl(shortUrl); err != nil {
		return 0, err
	}

	count, err := storeService.redisClient.Get(ctx, clicksKey(shortUrl)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed retrieving click count for %s: %w", shortUrl, err)
	}
	return count, nil
}
