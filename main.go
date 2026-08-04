package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	// r is the router/engine. gin.Default() creates a Gin engine with two middleware already attached
	// 1. a logger (prints req logs to console)
	// 2. recovery middleware (catches panics inside handlers so one bad req doesn't crash whole server)
	// Alternatively, gin.New() gives a bare engine to use
	r := gin.Default()

	// Register a GET route to "/"
	// The anonymous function is a handler and c is the context for a single incoming HTTP call.
	// The Context bundles together incoming request data, mechanism to write the response, and gin metadata.
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hey Go URL Shortener !",
		})
	})

	err := r.Run(":9808")
	if err != nil {
		panic(fmt.Sprintf("Failed to start the web server - Error: %v", err))
	}

}