package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"web-scraper-api/scrape"

	_ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/task_queue?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    payload := scrape.ScrapeRequest{
        URL: "https://blog.railway.com/p/launch-week-01-horizontal-scaling#%F0%9F%8E%BB-orchestrating-replicas",
        Text: true,
    }
    payloadBytes, _ := json.Marshal(payload)

    _, err = db.Exec("INSERT INTO tasks (payload) VALUES ($1)", payloadBytes)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Task enqueued successfully")
}