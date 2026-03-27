package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
	Status int         `json:"-"`
}

func RespondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response := Response{
		Data:   payload,
		Status: status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func RespondWithError(w http.ResponseWriter, status int, message string) {
	response := Response{
		Error:  message,
		Status: status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
