package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
)

// RateLimit returns Gin middleware that allows at most `limit` requests per
// client IP within `window`, backed by a Redis counter so the limit holds
// even if the backend runs as multiple replicas.
func RateLimit(store *store.StorageService, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())

		count, err := store.IncrementRateLimitCounter(c.Request.Context(), key, window)
		if err != nil {
			// Fail open: a Redis hiccup shouldn't take the whole endpoint down.
			log.Printf("rate limit check failed, allowing request through: %v", err)
			c.Next()
			return
		}

		if count > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded, try again later"})
			return
		}

		c.Next()
	}
}
