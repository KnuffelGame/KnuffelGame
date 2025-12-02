package middleware

import (
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// InternalOnly returns middleware that restricts access to internal routes.
// It checks the X-Internal-Token header against Config.InternalToken.
// If missing or mismatched, responds with 401 Unauthorized JSON.
func InternalOnly(cfg *models.Config) func(http.Handler) http.Handler {
	var token string
	if cfg != nil {
		token = cfg.InternalToken
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context())
			h := r.Header.Get("X-Internal-Token")
			if token == "" || h == "" || h != token {
				httpx.WriteUnauthorized(w, "Invalid internal token", log)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
