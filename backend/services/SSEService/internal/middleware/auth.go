package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// userCtxKey prevents context collisions in this package.
type userCtxKey struct{}

// UserContext holds minimal authenticated user information.
type UserContext struct {
	UserID   string
	Username string
}

// WithUserContext reads the user context from request context.
func WithUserContext(r *http.Request) (UserContext, bool) {
	if r == nil {
		return UserContext{}, false
	}
	if v, ok := r.Context().Value(userCtxKey{}).(UserContext); ok {
		return v, true
	}
	return UserContext{}, false
}

// AuthMiddleware validates JWT from cookie and injects user context and headers.
func AuthMiddleware(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
	cookieName := "jwt"
	if cfg != nil && cfg.JWTCookieName != "" {
		cookieName = cfg.JWTCookieName
	}
	validateURL := ""
	if cfg != nil && cfg.APIGatewayBaseURL != "" {
		// AuthService validate endpoint proxied via APIGateway
		validateURL = cfg.APIGatewayBaseURL + "/internal/validate"
	}
	client := &http.Client{Timeout: 3 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(
				slog.String("action", "auth"),
			)
			if baseLog != nil {
				log = baseLog.WithGroup("middleware").With(slog.String("action", "auth"))
			}

			c, err := r.Cookie(cookieName)
			if err != nil || c == nil || c.Value == "" {
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}

			if validateURL == "" {
				// Without validation endpoint, we cannot authenticate
				log.Error("APIGatewayBaseURL missing; cannot validate JWT")
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}

			body, _ := json.Marshal(map[string]string{"token": c.Value})
			req, err := http.NewRequest(http.MethodPost, validateURL, bytes.NewReader(body))
			if err != nil {
				log.Error("request build failed", slog.String("error", err.Error()))
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			// propagate internal token if configured
			if cfg != nil && cfg.InternalToken != "" {
				req.Header.Set("X-Internal-Token", cfg.InternalToken)
			}
			// correlate request id if present
			if rid := r.Header.Get("X-Request-ID"); rid != "" {
				req.Header.Set("X-Request-ID", rid)
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Error("token validation call failed", slog.String("error", err.Error()))
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}
			defer resp.Body.Close()

			var v struct {
				Valid    bool   `json:"valid"`
				UserID   string `json:"user_id"`
				Username string `json:"username"`
				IsGuest  bool   `json:"is_guest"`
				Error    string `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
				log.Warn("decode validate response failed", slog.String("error", err.Error()))
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}

			if resp.StatusCode != http.StatusOK || !v.Valid || v.UserID == "" || v.Username == "" {
				log.Info("authentication failed",
					slog.Int("status", resp.StatusCode),
					slog.String("error", v.Error),
				)
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}

			// success: inject headers and context
			r.Header.Set("X-User-ID", v.UserID)
			r.Header.Set("X-Username", v.Username)
			ctx := context.WithValue(r.Context(), userCtxKey{}, UserContext{UserID: v.UserID, Username: v.Username})
			log.Info("authenticated", slog.String("user_id", v.UserID), slog.String("username", v.Username))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
