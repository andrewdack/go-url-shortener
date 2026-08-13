package router

import (
	"time"

	"github.com/andrewdack/go-url-shortener/handler"
	"github.com/andrewdack/go-url-shortener/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter builds the Gin engine and registers all routes.
func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	// Rate limited to 10 requests per minute per client IP
	r.POST("/create-short-url", middleware.RateLimit(10, time.Minute), handler.CreateShortUrl)
	r.GET("/:shortUrl", middleware.ClickCounter, handler.HandleShortUrlRedirect)
	r.GET("/:shortUrl/count", handler.GetShortUrlClicks)
	r.PATCH("/:shortUrl", handler.UpdateShortUrl)
	r.DELETE("/:shortUrl", handler.DeleteShortUrl)

	return r
}
