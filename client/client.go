package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
	"web-scraper-api/queue"
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

	httpreq, err := http.NewRequest("POST", c.URL+"/scrape", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.New("failed to create request: " + err.Error())
	}
	httpreq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpreq)
	if err != nil {
		return nil, errors.New("failed to send request: " + err.Error())
	}
	defer resp.Body.Close()

	var taskResponse queue.CreateTaskResponse
	err = json.NewDecoder(resp.Body).Decode(&taskResponse)
	if err != nil {
		return nil, errors.New("failed to decode response: " + err.Error())
	}

	//fmt.Println("Task created with ID:", taskResponse.ID)

	for range time.Tick(1 * time.Second) {
		httpreq, err := http.NewRequest("GET", c.URL+"/scrape/"+strconv.Itoa(taskResponse.ID), nil)
		if err != nil {
			return nil, errors.New("failed to create request for scrape result: " + err.Error())
		}
		resp, err := http.DefaultClient.Do(httpreq)
		if err != nil {
			return nil, errors.New("failed to send request for scrape result: " + err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			taskResponse := queue.Task{}
			err = json.NewDecoder(resp.Body).Decode(&taskResponse)
			if err != nil {
				return nil, errors.New("failed to decode response with scrape result: " + err.Error())
			}
			//fmt.Printf("%+v\n", taskResponse)
			if taskResponse.Status == "done" {
				err = json.Unmarshal(taskResponse.Result, &page)
				if err != nil {
					return nil, errors.New("failed to decode scrape result: " + err.Error())
				}
				break
			}
		}
	}

	return page, nil
}
