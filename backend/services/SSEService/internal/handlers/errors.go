package handlers

import (
	"net/http"

	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
)

// BadRequest writes a 400 Bad Request error with optional details map.
func BadRequest(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
	httpx.WriteBadRequest(w, message, details, log)
}

// Unauthorized writes a 401 Unauthorized error.
func Unauthorized(w http.ResponseWriter, message string, log *slog.Logger) {
	httpx.WriteUnauthorized(w, message, log)
}

// Forbidden writes a 403 Forbidden error.
func Forbidden(w http.ResponseWriter, message string, log *slog.Logger) {
	httpx.WriteForbidden(w, message, log)
}

// NotFound writes a 404 Not Found error.
func NotFound(w http.ResponseWriter, message string, log *slog.Logger) {
	httpx.WriteNotFound(w, message, log)
}

// Conflict writes a 409 Conflict error.
func Conflict(w http.ResponseWriter, message string, log *slog.Logger) {
	httpx.WriteError(w, http.StatusConflict, "conflict", message, nil, log)
}

// InternalServerError writes a 500 Internal Server Error with optional details map.
func InternalServerError(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
	httpx.WriteInternalError(w, message, details, log)
}
