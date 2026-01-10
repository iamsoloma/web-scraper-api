package scrape

import "time"

type ScrapeRequest struct {
	URL        string `json:"url"`
	Screenshot bool   `json:"screenshot"`
	HTML       bool   `json:"html"`
	Markdown   bool   `json:"markdown"`
	Text       bool   `json:"text"`
	Images     bool   `json:"images"`
	Links      bool   `json:"links"`
}

type ScraperResponse struct {
	Title      string            `json:"title"`
	Screenshot string            `json:"screenshot"`
	HTML       string            `json:"html"`
	Markdown   string            `json:"markdown"`
	Text       string            `json:"text"`
	Images     map[string]string `json:"images"`
	Links      map[string]string `json:"links"`
	Date       time.Time         `json:"date"`
}
