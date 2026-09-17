package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"web-scraper-api/config"
	"web-scraper-api/queue"
	"web-scraper-api/storage"
)

type Server struct {
	Config  *config.Config
	Started time.Time
	Version string
	Queue   *queue.Queue
	Storage *storage.Storage
}

func NewServer(config config.Config, version string) (*Server, error) {
	queue, err := queue.NewPostgressConnect(config.Database)
	if err != nil {
		return nil, errors.New("can`t connect to postgres: " + err.Error())
	}

	err = queue.CreateTaskTable()
	if err != nil {
		return nil, errors.New("can`t create task table: " + err.Error())
	}

	storage := storage.NewStorage(config.Storage)

	return &Server{
		Config:  &config,
		Version: version,
		Queue:   queue,
		Storage: storage,
	}, nil
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("POST /scrape", s.Scrape)
	mux.HandleFunc("GET /scrape/{id}", s.ScrapeResult)

	mainHandler := NewLoggerMiddleware(mux)

	server := http.Server{
		Addr:    s.Config.ListenAddr,
		Handler: mainHandler,
	}

	s.Started = time.Now().UTC()

	go s.Queue.HandleFunc("scrape", s.Scraper, s.Storage)

	slog.Info("api is running", "address", s.Config.ListenAddr)
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("API stoped", "error", err)
	}
}
