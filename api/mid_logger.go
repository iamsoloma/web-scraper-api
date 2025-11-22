package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type LogMessage struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Latency string `json:"latency"`
}

type LoggerMiddleware struct {
	handler http.Handler
}

func (l *LoggerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	jsonMessage, err := json.Marshal(LogMessage{Method: r.Method, Path: r.URL.Path, Latency: time.Since(start).String()})
	if err != nil {
		log.Println("Error marshaling log message:", err)
		return
	}
	fmt.Println(string(jsonMessage))
}

func NewLoggerMiddleware(handler http.Handler) http.Handler {
	return &LoggerMiddleware{handler: handler}
}