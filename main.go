package main

import (
	"fmt"

	"github.com/andrewdack/go-url-shortener/router"
	"github.com/andrewdack/go-url-shortener/store"
)

func main() {
	// Note that store initialization happens here
	store.InitializeStore()

	r := router.SetupRouter()

	err := r.Run(":9808")
	if err != nil {
		panic(fmt.Sprintf("Failed to start the web server - Error: %v", err))
	}
}
