package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type Health struct {
	Status      string `json:"Status"`
	CurrentTime string `json:"CurrentTime"`
	Uptime      string `json:"Uptime"`
	Version     string `json:"BackendVersion"`
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {

	jsonResp, err := json.Marshal(Health{
		Status:      "Ok",
		CurrentTime: time.Now().UTC().String(),
		Uptime:      time.Since(s.Started).String(),
		Version:     "0.0.4",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}