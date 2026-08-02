package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"web-scraper-api/queue"
	"web-scraper-api/scrape"
)

func (s *Server) Scrape(w http.ResponseWriter, r *http.Request) {
	var scrapeReq scrape.ScrapeRequest
	err := json.NewDecoder(r.Body).Decode(&scrapeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	scrapeBytes, err := json.Marshal(scrapeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}

	id, err := s.Queue.CreateTask("scrape", scrapeBytes, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create task: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(queue.CreateTaskResponse{ID: id})
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *Server) Scraper(payload []byte) (result []byte, err error) {
	var scrapeReq scrape.ScrapeRequest
	err = json.Unmarshal(payload, &scrapeReq)
	if err != nil {
		return nil, errors.New("Invalid request body: " + err.Error())
	}

	requrl, err := url.Parse(scrapeReq.URL)
	if err != nil {
		return nil, errors.New("URL parse: " + err.Error())
	}

	robotsParser, err := scrape.NewRobotsParser(scrapeReq.URL)
	if err != nil {
		return nil, errors.New("Robots parser: " + err.Error())
	}

	err = robotsParser.Fetch()
	if err != nil {
		return nil, errors.New("Robots fetch: " + err.Error())
	}
	allowed := robotsParser.IsAllowed(s.Config.UserAgent, requrl.Path)
	if !allowed {
		return nil, errors.New("Scraping disallowed by robots.txt")
	}

	scrapeResp, err := scrape.Scrape(scrapeReq, s.Config.UserAgent)
	if err != nil {
		return nil, errors.New("Scraping: " + err.Error())
	}

	result, err = json.Marshal(scrapeResp)
	if err != nil {
		return nil, errors.New("JSON: " + err.Error())
	}
	return result, nil
}

func (s *Server) ScrapeResult(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	id, err := strconv.Atoi(sid)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}
	task, err := s.Queue.GetTask(id)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(task)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
