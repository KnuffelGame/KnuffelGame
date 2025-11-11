package router

import (
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/go-chi/chi/v5"
)

// New constructs the HTTP router and attaches logging middleware.
// Currently returns basic HTTP server setup for SSE service.
// TODO: Add SSE-specific routes and handlers
func New() http.Handler {
	r := chi.NewRouter()
	// Replace chi default logger with structured slog based middleware
	l := logger.Default()
	r.Use(logger.ChiMiddleware(l))

	// Healthcheck
	healthcheck.Mount(r)

	// Routes
	// TODO: Add SSE event routes
	// GET /events/lobby/{id} -> subscribe to lobby events
	// GET /events/game/{id} -> subscribe to game events
	// POST /internal/publish -> internal publish endpoint

	// Temporary stub for testing
	r.Get("/internal/test", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}, logger.Logger(r.Context()))
	})

	return r
}
