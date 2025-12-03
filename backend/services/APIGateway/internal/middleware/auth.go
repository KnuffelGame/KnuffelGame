package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/pkg/config"
)

type contextKey string

const (
	userIDKey   contextKey = "userID"
	usernameKey contextKey = "username"
)

// Authentication middleware handles JWT validation and creation workflows.
func Authentication(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			method := r.Method

			if method == "POST" && (path == "/lobbies" || path == "/lobbies/join") {
				// Workflow 2: Token creation
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read body", http.StatusBadRequest)
					return
				}

				var req struct {
					Username string `json:"username"`
				}
				if err := json.Unmarshal(body, &req); err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
					return
				}

				// Reset body for proxy
				r.Body = io.NopCloser(bytes.NewReader(body))

				// Call AuthService /internal/create
				client := &http.Client{}
				authReqBody := map[string]string{"username": req.Username}
				authReqJSON, _ := json.Marshal(authReqBody)
				authReq, err := http.NewRequest("POST", cfg.AuthServiceURL+"/internal/create", bytes.NewReader(authReqJSON))
				if err != nil {
					http.Error(w, "Failed to create auth request", http.StatusInternalServerError)
					return
				}
				authReq.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(authReq)
				if err != nil {
					http.Error(w, "Auth service error", http.StatusInternalServerError)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					http.Error(w, "Failed to create token", resp.StatusCode)
					return
				}

				var authResp struct {
					Token    string `json:"token"`
					Username string `json:"username"`
					UserID   string `json:"user_id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
					http.Error(w, "Invalid auth response", http.StatusInternalServerError)
					return
				}

				// Set to context
				ctx := context.WithValue(r.Context(), userIDKey, authResp.UserID)
				ctx = context.WithValue(ctx, usernameKey, authResp.Username)
				r = r.WithContext(ctx)

				// Wrap response to set cookie
				wrapped := &responseWrapper{
					ResponseWriter: w,
					token:          authResp.Token,
					cfg:            cfg,
				}
				next.ServeHTTP(wrapped, r)
			} else {
				// Workflow 1: Token validation
				cookie, err := r.Cookie("jwt")
				if err != nil {
					http.Error(w, "No token", http.StatusUnauthorized)
					return
				}

				token := cookie.Value

				// Call AuthService /internal/validate
				client := &http.Client{}
				authReqBody := map[string]string{"token": token}
				authReqJSON, _ := json.Marshal(authReqBody)
				authReq, err := http.NewRequest("POST", cfg.AuthServiceURL+"/internal/validate", bytes.NewReader(authReqJSON))
				if err != nil {
					http.Error(w, "Failed to create auth request", http.StatusInternalServerError)
					return
				}
				authReq.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(authReq)
				if err != nil {
					http.Error(w, "Auth service error", http.StatusInternalServerError)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}

				var authResp struct {
					Valid    bool   `json:"valid"`
					UserID   string `json:"user_id"`
					Username string `json:"username"`
					IsGuest  bool   `json:"is_guest"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil || !authResp.Valid {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}

				// Set to context
				ctx := context.WithValue(r.Context(), userIDKey, authResp.UserID)
				ctx = context.WithValue(ctx, usernameKey, authResp.Username)
				r = r.WithContext(ctx)

				next.ServeHTTP(w, r)
			}
		})
	}
}

// HeaderInjection middleware sets X-User-ID and X-Username headers from context.
func HeaderInjection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if userID, ok := ctx.Value(userIDKey).(string); ok {
			r.Header.Set("X-User-ID", userID)
		}
		if username, ok := ctx.Value(usernameKey).(string); ok {
			r.Header.Set("X-Username", username)
		}
		next.ServeHTTP(w, r)
	})
}

// responseWrapper wraps the ResponseWriter to add Set-Cookie header.
type responseWrapper struct {
	http.ResponseWriter
	token         string
	cfg           *config.Config
	headerWritten bool
}

func (rw *responseWrapper) WriteHeader(statusCode int) {
	if !rw.headerWritten {
		cookie := &http.Cookie{
			Name:     "jwt",
			Value:    rw.token,
			Domain:   rw.cfg.CookieDomain,
			Path:     "/",
			HttpOnly: true,
			Secure:   rw.cfg.CookieSecure,
		}
		switch rw.cfg.CookieSameSite {
		case "Strict":
			cookie.SameSite = 3
		case "Lax":
			cookie.SameSite = 2
		case "None":
			cookie.SameSite = 4
		default:
			cookie.SameSite = 2
		}
		rw.Header().Add("Set-Cookie", cookie.String())
		rw.headerWritten = true
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWrapper) Write(data []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}
