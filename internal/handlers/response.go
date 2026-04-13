package handlers

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Data any `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"` // For validation errors
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func respondWithData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, Response{
		Data: data,
	})
}

func respondWithError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{
		Error: err.Error(),
	})
}

func respondWithValidationError(w http.ResponseWriter, details any) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   errValidationFailed.Error(),
		Details: details,
	})
}
