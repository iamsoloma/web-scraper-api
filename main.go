package main

import (
	"log"
	"os"
	"web-scraper-api/api"
	"web-scraper-api/config"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		userAgent = "OpinionBot"
	}

	config := config.MustLoadConf()

	server, err := api.NewServer(config, "0.0.7")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	server.Start()
}
