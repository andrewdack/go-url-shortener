package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
)

// ClickCounter increments a per-shortUrl click count, but only after a
// successful redirect — so a 404 on a dead link isn't counted as a click.
func ClickCounter(c *gin.Context) {
	// Because we only increment counter on successful redirect, handle the route first
	c.Next()

	// Redirect failed, don't increment
	if c.Writer.Status() != http.StatusFound {
		return
	}

	shortUrl := c.Param("shortUrl")
	key := fmt.Sprintf("clicks:%s", shortUrl)
	if _, err := store.IncrementClickCount(key, shortUrl); err != nil {
		log.Printf("failed to increment click count for %s: %v", shortUrl, err)
	}
}
