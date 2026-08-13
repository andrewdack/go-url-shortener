package router

import (
	"time"

	"github.com/andrewdack/go-url-shortener/handler"
	"github.com/andrewdack/go-url-shortener/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter builds the Gin engine and registers all routes.
func SetupRouter(h *handler.Handler) *gin.Engine {
	r := gin.Default()

	// Apply CORS middleware to all routes
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})

	r.POST("/create-short-url", middleware.RateLimit(h.Store(), 10, time.Minute), h.CreateShortUrl) // Rate limited to 10 req per minute per IP
	r.GET("/:shortUrl", middleware.ClickCounter(h.Store()), h.HandleShortUrlRedirect)
	r.GET("/:shortUrl/count", h.GetShortUrlClicks)
	r.PATCH("/:shortUrl", h.UpdateShortUrl)
	r.DELETE("/:shortUrl", h.DeleteShortUrl)

	return r
}
