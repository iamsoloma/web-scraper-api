package main

import (
	"log"
	"os"
	"web-scraper-api/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}

	server, err := api.NewServer(api.Config{
		ListenAddr: port,
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	server.Start()
}
