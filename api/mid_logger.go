package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type LogMessage struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Query   string `json:"query"`
	Body    string `json:"body"`
	Latency string `json:"latency"`
}

type LoggerMiddleware struct {
	handler http.Handler
}

func (l *LoggerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading request body:", err)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	jsonMessage, err := json.Marshal(LogMessage{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body), Latency: time.Since(start).String()})
	if err != nil {
		log.Println("Error marshaling log message:", err)
		return
	}
	fmt.Println(string(jsonMessage))
}

func NewLoggerMiddleware(handler http.Handler) http.Handler {
	return &LoggerMiddleware{handler: handler}
}
