package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"web-scraper-api/client"
	"web-scraper-api/scrape"
)

func main() {

	//Create a new client instance pointing to the local server
	client := client.NewClient("http://localhost:8080")

	//Make a scrape request with the desired URL and options
	page, err := client.Scrape(&scrape.ScrapeRequest{
		URL:        "https://en.wikipedia.org/wiki/War_and_Peace",
		Screenshot: true,
		Markdown:   true,
	})
	if err != nil {
		fmt.Println("Error scraping page:", err)
	}

	//Print the scraped page title and markdown content
	println("Title:", page.Title)
	println("Markdown Content:", page.Markdown)
	//Decode the base64 screenshot data
	ScreenBytes, err := base64.StdEncoding.DecodeString(page.Screenshot)
	if err != nil {
		fmt.Println("Error decoding screenshot:", err)
	}
	//Save the screenshot to a file
	err = os.WriteFile("./screenshot.png", ScreenBytes, 0644)
	if err != nil {
		fmt.Println("Error saving screenshot:", err)
	}
}
