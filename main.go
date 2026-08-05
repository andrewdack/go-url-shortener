package main

import (
	"fmt"

	"github.com/andrewdack/go-url-shortener/handler"
	"github.com/andrewdack/go-url-shortener/store"
	"github.com/gin-gonic/gin"
)

// Main function to start the web server and register routes
// Note: In more complex apps, endpoints should live in a separate file
// For sake of simplicity, we are keeping them here for now
func main() {
	// r is the router/engine. gin.Default() creates a Gin engine with two middleware already attached
	// 1. a logger (prints req logs to console)
	// 2. recovery middleware (catches panics inside handlers so one bad req doesn't crash whole server)
	// Alternatively, gin.New() gives a bare engine to use
	r := gin.Default()

	// Health check route
	// Register a GET route to "/"
	// The anonymous function is a handler and c is the context for a single incoming HTTP call.
	// The Context bundles together incoming request data, mechanism to write the response, and gin metadata.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})


	// Register a POST route to "/create-short-url"
	// Pass the request context to the handler function
	r.POST("/create-short-url", func(c *gin.Context) {
		handler.CreateShortUrl(c)
	})

	r.GET("/:shortUrl", func(c *gin.Context) {
		handler.HandleShortUrlRedirect(c)
	})

	// Note that store initialization happens here
	store.InitializeStore()

	err := r.Run(":9808")
	if err != nil {
		panic(fmt.Sprintf("Failed to start the web server - Error: %v", err))
	}

}
