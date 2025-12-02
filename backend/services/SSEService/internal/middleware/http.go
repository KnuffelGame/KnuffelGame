package middleware

import (
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// CORS returns a chi-compatible middleware configured from Config.
func CORS(cfg *models.Config) func(http.Handler) http.Handler {
	if cfg == nil {
		return nil
	}
	return cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowOrigins,
		AllowedMethods:   cfg.CORSAllowMethods,
		AllowedHeaders:   cfg.CORSAllowHeaders,
		AllowCredentials: cfg.CORSAllowCredentials,
		// We keep defaults for MaxAge and ExposedHeaders for now
	})
}

// RateLimitInternal returns a per-IP rate limiter for internal endpoints.
// If RateLimitInternalPerMinute <= 0, returns nil (disabled).
func RateLimitInternal(cfg *models.Config) func(http.Handler) http.Handler {
	if cfg == nil || cfg.RateLimitInternalPerMinute <= 0 {
		return nil
	}
	return httprate.LimitByIP(cfg.RateLimitInternalPerMinute, 1*time.Minute)
}

// RateLimitSSE returns a per-IP rate limiter for SSE endpoints.
// If RateLimitSSEPerMinute == 0, returns nil (disabled/recommended for long-lived connections).
func RateLimitSSE(cfg *models.Config) func(http.Handler) http.Handler {
	if cfg == nil || cfg.RateLimitSSEPerMinute == 0 {
		return nil
	}
	if cfg.RateLimitSSEPerMinute < 0 {
		// negative values also considered disabled
		return nil
	}
	return httprate.LimitByIP(cfg.RateLimitSSEPerMinute, 1*time.Minute)
}
