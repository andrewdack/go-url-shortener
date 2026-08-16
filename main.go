package main

import (
	"context"
	"log"
	"os"

	"github.com/andrewdack/go-url-shortener/handler"
	"github.com/andrewdack/go-url-shortener/router"
	"github.com/andrewdack/go-url-shortener/store"
)

func main() {
	// Create context
	ctx := context.Background()

	// Note that store initialization happens here
	storeService, err := store.NewStore(ctx)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer storeService.Close()

	h := handler.NewHandler(storeService)
	r := router.SetupRouter(h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9808"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
