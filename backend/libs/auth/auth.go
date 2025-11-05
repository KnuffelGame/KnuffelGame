package auth

import (
	"context"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/google/uuid"
	"log/slog"
)

const (
	DefaultHeaderUserID   = "X-User-ID"
	DefaultHeaderUsername = "X-Username"
)

// ctxKey is an unexported type for context keys in this package.
type ctxKey struct{}

// User represents an authenticated user extracted from gateway headers.
type User struct {
	ID       uuid.UUID
	Username string
}

// FromContext returns the User stored in context (if any).
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}

// NewAuthMiddleware constructs an auth middleware that reads the given header names and injects a User into context.
// It returns a chi/standard middleware: func(next http.Handler) http.Handler
func NewAuthMiddleware(userHeader, usernameHeader string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(slog.String("action", "auth"))

			userIDStr := r.Header.Get(userHeader)
			username := r.Header.Get(usernameHeader)

			if userIDStr == "" || username == "" {
				log.Warn("missing auth headers", slog.String("user_id", userIDStr), slog.String("username", username))
				httpx.WriteBadRequest(w, "Missing required headers: X-User-ID and X-Username", nil, log)
				return
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				log.Warn("invalid user_id format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
				httpx.WriteBadRequest(w, "Invalid user ID format", map[string]interface{}{"detail": err.Error()}, log)
				return
			}

			u := User{ID: userID, Username: username}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
		})
	}
}

// Default middleware using standard header names
var AuthMiddleware = NewAuthMiddleware(DefaultHeaderUserID, DefaultHeaderUsername)
