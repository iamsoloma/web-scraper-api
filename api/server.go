package api

import (
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	ListenAddr string
	UserAgent  string
}

type Server struct {
	*Config
	Started time.Time
	Version string
}

func NewServer(config Config, version string) (*Server, error) {
	return &Server{
		Config:  &config,
		Version: version,
	}, nil
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /scrape", s.Scrape)

	mainHandler := NewLoggerMiddleware(mux)

	server := http.Server{
		Addr:    s.Config.ListenAddr,
		Handler: mainHandler,
	}

	s.Started = time.Now().UTC()

	slog.Info("api is running", "address", s.Config.ListenAddr)
	err := server.ListenAndServe()
	if err != nil {
		slog.Error("API stoped", "error", err)
	}

}
