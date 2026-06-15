package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"web-scraper-api/scrape"
)

type Client struct {
	URL string
}

func NewClient(url string) *Client {
	return &Client{URL: url}
}

func (c *Client) Scrape(req *scrape.ScrapeRequest) (page *scrape.ScraperResponse, err error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpreq, err := http.NewRequest("GET", c.URL+"/scrape", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	httpreq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&page)
	if err != nil {
		return nil, err
	}

	return page, nil
}
