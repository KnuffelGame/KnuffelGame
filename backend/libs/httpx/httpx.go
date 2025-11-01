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

// ErrorPayload standard error response envelope.
// Error is a machine readable string, Message human readable, Details optional map.
type ErrorPayload struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
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

// WriteError wraps WriteJSON for error responses.
func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}, log *slog.Logger) {
	WriteJSON(w, status, ErrorPayload{Error: code, Message: message, Details: details}, log)
}

// WriteBadRequest is a convenience wrapper for 400 Bad Request responses.
func WriteBadRequest(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
	WriteError(w, http.StatusBadRequest, "bad_request", message, details, log)
}

// WriteUnauthorized is a convenience wrapper for 401 Unauthorized responses.
func WriteUnauthorized(w http.ResponseWriter, message string, log *slog.Logger) {
	WriteError(w, http.StatusUnauthorized, "unauthorized", message, nil, log)
}

// WriteForbidden is a convenience wrapper for 403 Forbidden responses.
func WriteForbidden(w http.ResponseWriter, message string, log *slog.Logger) {
	WriteError(w, http.StatusForbidden, "forbidden", message, nil, log)
}

// WriteNotFound is a convenience wrapper for 404 Not Found responses.
func WriteNotFound(w http.ResponseWriter, message string, log *slog.Logger) {
	WriteError(w, http.StatusNotFound, "not_found", message, nil, log)
}

// WriteInternalError is a convenience wrapper for 500 Internal Server Error responses.
func WriteInternalError(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
	WriteError(w, http.StatusInternalServerError, "internal_error", message, details, log)
}

// WriteNoContent writes a 204 No Content response.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// DecodeJSON decodes the request body as JSON into the provided target.
// Returns an error if the body is invalid JSON or cannot be decoded.
func DecodeJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return json.Unmarshal([]byte("{}"), target)
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
