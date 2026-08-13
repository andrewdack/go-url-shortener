package middleware

import (
	"log"
	"net/http"

	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
)

// ClickCounter returns middleware that increments a per-shortUrl click count
// only after a successful redirect.
func ClickCounter(storeService *store.StorageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() != http.StatusFound {
			return
		}

		shortUrl := c.Param("shortUrl")
		if _, err := storeService.IncrementClickCount(c.Request.Context(), shortUrl); err != nil {
			log.Printf("failed to increment click count for %s: %v", shortUrl, err)
		}
	}
}
