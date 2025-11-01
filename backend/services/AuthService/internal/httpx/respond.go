package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// LoggerProvider abstracts obtaining a request-scoped logger.
// Allows decoupling from the logger package for easier testing.
type LoggerProvider interface {
	Logger() *slog.Logger
}

// WriteJSON writes the provided payload as JSON with the given status code.
// If encoding fails it logs and sends a generic 500 error JSON.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		if log != nil {
			log.Error("json encode failed", slog.String("error", err.Error()))
		}
		// Best effort fallback; can't change status now.
	}
}

// ErrorPayload standard error response envelope.
// Code is a machine readable string, Message human readable, Details optional map.

type ErrorPayload struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// WriteError wraps WriteJSON for error responses.
func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}, log *slog.Logger) {
	WriteJSON(w, status, ErrorPayload{Error: code, Message: message, Details: details}, log)
}
