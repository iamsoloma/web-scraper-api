package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"web-scraper-api/config"
	"web-scraper-api/scrape"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateTaskResponse struct {
	ID int `json:"taskID"`
}

type Queue struct {
	pool *pgxpool.Pool
}

type ResultStorage interface {
	PutObject(key string, data []byte) error
}

type Task struct {
	ID            int             `json:"id"`
	Task          string          `json:"task"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	Priority      int             `json:"priority"`
	Attempts      int             `json:"attempts"`
	CreatedAt     time.Time       `json:"created_at"`
	LastAttemptAt time.Time       `json:"last_attempt_at"`
	Result        json.RawMessage `json:"result"`
	ResultKey     string          `json:"-"`
}

func NewPostgressConnect(config config.Database) (*Queue, error) {
	DatabaseUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		config.User,
		config.Password,
		config.Domain,
		config.Port,
		config.DBName)

	pool, err := pgxpool.New(context.Background(), DatabaseUrl)
	if err != nil {
		return nil, err
	}
	return &Queue{pool: pool}, nil
}

func (q *Queue) Close() {
	q.pool.Close()
}

func (q *Queue) CreateTaskTable() error {
	_, err := q.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS tasks (
    		id SERIAL PRIMARY KEY,
			task VARCHAR(255) NOT NULL,
    		payload JSONB NOT NULL,
    		status VARCHAR(20) NOT NULL DEFAULT 'pending',
			priority INT NOT NULL DEFAULT 0,
    		attempts INT NOT NULL DEFAULT 0,
    		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    		last_attempt_at TIMESTAMP WITH TIME ZONE,
			result_key VARCHAR(255)
		);
	`)
	if err != nil {
		return err
	}

	return err
}

func (q *Queue) CreateTask(task string, payload []byte, priority int) (id int, err error) {
	err = q.pool.QueryRow(context.Background(), `
		INSERT INTO tasks (task, payload, priority)
		VALUES ($1, $2, $3)
		RETURNING id;
	`, task, payload, priority).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (q *Queue) GetTask(id int) (task *Task, err error) {
	task = &Task{}
	err = q.pool.QueryRow(context.Background(), `
		SELECT id, task, payload, status, priority, attempts, created_at, last_attempt_at, COALESCE(result_key, '')
		FROM tasks
		WHERE id = $1;
	`, id).Scan(&task.ID, &task.Task, &task.Payload, &task.Status, &task.Priority, &task.Attempts, &task.CreatedAt, &task.LastAttemptAt, &task.ResultKey)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (q *Queue) HandleFunc(task string, handler func(payload []byte) (result []byte, err error), resultStorage ResultStorage) {
	for {
		ctx := context.Background()
		tx, err := q.pool.Begin(ctx)
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}

		var id int
		var payloadBytes []byte
		err = tx.QueryRow(ctx, `SELECT id, payload FROM tasks 
            WHERE task = $1 AND status = 'pending' 
            ORDER BY priority DESC, created_at ASC 
            FOR UPDATE SKIP LOCKED 
            LIMIT 1`, task).Scan(&id, &payloadBytes)
		if err == pgx.ErrNoRows {
			tx.Rollback(ctx)
			time.Sleep(1 * time.Second)
			continue
		} else if err != nil {
			tx.Rollback(ctx)
			log.Println(err)
			continue
		}

		_, err = tx.Exec(ctx, "UPDATE tasks SET status = 'processing', last_attempt_at = CURRENT_TIMESTAMP WHERE id = $1", id)
		if err != nil {
			tx.Rollback(ctx)
			log.Println(err)
			continue
		}

		tx.Commit(ctx)

		var task scrape.ScrapeRequest
		err = json.Unmarshal(payloadBytes, &task)
		// Process the task here, e.g., send email based on payload["action"]
		if err != nil {
			_, err = q.pool.Exec(ctx, "UPDATE tasks SET status = 'failed' WHERE id = $1", id)
		}

		result, err := handler(payloadBytes)

		if err != nil {
			attempts := 1      // Fetch actual attempts from query
			if attempts >= 3 { // Max retries
				_, err = q.pool.Exec(ctx, "UPDATE tasks SET status = 'failed' WHERE id = $1", id)
			} else {
				_, err = q.pool.Exec(ctx, "UPDATE tasks SET status = 'pending', attempts = attempts + 1 WHERE id = $1", id)
			}
			if err != nil {
				log.Println(err)
			}
		} else {
			resultKey := fmt.Sprintf("results/%d.json", id)
			err = resultStorage.PutObject(resultKey, result)
			if err != nil {
				log.Println(err)
				continue
			}

			_, err = q.pool.Exec(ctx, "UPDATE tasks SET status = 'done', result_key = $2 WHERE id = $1", id, resultKey)
			if err != nil {
				log.Println(err)
			}
		}

		fmt.Printf("Processed task %d\n", id)
	}
}
