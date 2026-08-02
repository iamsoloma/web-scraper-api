package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"web-scraper-api/scrape"
)

func (s *Server) Scrape(w http.ResponseWriter, r *http.Request) {
	var scrapeReq scrape.ScrapeRequest
	err := json.NewDecoder(r.Body).Decode(&scrapeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	requrl, err := url.Parse(scrapeReq.URL)
	if err != nil {
		http.Error(w, errors.New("URL parse: "+err.Error()).Error(), http.StatusBadRequest)
		return
	}

	robotsParser, err := scrape.NewRobotsParser(scrapeReq.URL)
	if err != nil {
		http.Error(w, errors.New("Robots parser: " + err.Error()).Error(), http.StatusInternalServerError)
		return 
	}

	err = robotsParser.Fetch()
	if err != nil {
		http.Error(w, errors.New("Robots fetch: " + err.Error()).Error(), http.StatusInternalServerError)
		return 
	}
	allowed := robotsParser.IsAllowed(s.Config.UserAgent, requrl.Path)
	if !allowed {
		http.Error(w, errors.New("Scraping disallowed by robots.txt").Error(), http.StatusForbidden)
		return
	}

	scrapeResp, err := scrape.Scrape(scrapeReq, s.Config.UserAgent)
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
