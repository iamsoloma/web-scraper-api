package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"web-scraper-api/scrape"
)

func (s *Server) Scrape(w http.ResponseWriter, r *http.Request) {
	var scrapeReq scrape.ScrapeRequest
	err := json.NewDecoder(r.Body).Decode(&scrapeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	scrapeResp, err := scrape.Scrape(scrapeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Scraping: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(scrapeResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}

}