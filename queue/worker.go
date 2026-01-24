package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"web-scraper-api/scrape"

	_ "github.com/lib/pq"
)

type Task struct {
	ID      int
	Payload map[string]string
}

func main() {
	db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/task_queue?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for {
		tx, err := db.Begin()
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}

		var id int
		var payloadBytes []byte
		err = tx.QueryRow(`SELECT id, payload FROM tasks 
            WHERE status = 'pending' 
            ORDER BY created_at ASC 
            FOR UPDATE SKIP LOCKED 
            LIMIT 1`).Scan(&id, &payloadBytes)
		if err == sql.ErrNoRows {
			tx.Rollback()
			time.Sleep(1 * time.Second)
			continue
		} else if err != nil {
			tx.Rollback()
			log.Println(err)
			continue
		}

		_, err = tx.Exec("UPDATE tasks SET status = 'processing', last_attempt_at = CURRENT_TIMESTAMP WHERE id = $1", id)
		if err != nil {
			tx.Rollback()
			log.Println(err)
			continue
		}

		tx.Commit()

		var task scrape.ScrapeRequest
		err = json.Unmarshal(payloadBytes, &task)
		// Process the task here, e.g., send email based on payload["action"]
		if err != nil {
			_, err = db.Exec("UPDATE tasks SET status = 'failed' WHERE id = $1", id)
		}

		scrapeResp, err := scrape.Scrape(task, "OpinionBot")
		fmt.Println(scrapeResp)

		if err != nil {
			attempts := 1      // Fetch actual attempts from query
			if attempts >= 3 { // Max retries
				_, err = db.Exec("UPDATE tasks SET status = 'failed' WHERE id = $1", id)
			} else {
				_, err = db.Exec("UPDATE tasks SET status = 'pending', attempts = attempts + 1 WHERE id = $1", id)
			}
			if err != nil {
				log.Println(err)
			}
		} else {
			_, err = db.Exec("UPDATE tasks SET status = 'done' WHERE id = $1", id)
			if err != nil {
				log.Println(err)
			}
		}

		fmt.Printf("Processed task %d\n", id)
		// Example output: Processed task 1
	}
}
