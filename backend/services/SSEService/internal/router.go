package router

import (
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	ssemw "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/middleware"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// New constructs the HTTP router, applies middleware, mounts healthcheck,
// and wires public SSE and protected internal routes (stubs for now).
func New(cfg *models.Config, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Request ID and structured logging middleware
	r.Use(chimw.RequestID)
	if log == nil {
		log = logger.Default()
	}
	r.Use(logger.ChiMiddleware(log))

	// CORS
	if mw := ssemw.CORS(cfg); mw != nil {
		r.Use(mw)
	}

	// Healthcheck
	healthcheck.Mount(r)

	// Registry singleton for handlers
	reg := models.NewRegistry()

	// Public SSE endpoints - rate limits disabled in MVP
	r.Route("/events", func(r chi.Router) {
		// Authenticated SSE endpoints with membership authorization
		r.Group(func(r chi.Router) {
			r.Use(ssemw.AuthMiddleware(cfg, log))
			r.Use(ssemw.AuthorizeLobbyMembership(cfg, log))
			r.Get("/lobby/{lobby_id}", handlers.SubscribeLobbyEvents(reg, cfg, log))
		})
	})

	// Internal endpoints - no token check per spec (isolated by reverse proxy)
	r.Route("/internal", func(r chi.Router) {
		// Internal endpoints using the registry
		r.Post("/publish", handlers.Publish(reg, log))
	})

	return r
}

// sseNotImplemented returns a stub SSE handler that sets standard SSE headers
// and returns 501 Not Implemented in JSON using httpx (until implemented).
func sseNotImplemented() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Standard SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Stub JSON error (httpx will set application/json Content-Type)
		httpx.WriteError(w, http.StatusNotImplemented, "not_implemented", "SSE streaming not implemented yet", nil, logger.Logger(r.Context()))
	}
}

// notImplemented returns a stub handler that responds with 501 JSON.
func notImplemented(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, http.StatusNotImplemented, "not_implemented", "Endpoint not implemented", map[string]interface{}{"endpoint": name}, logger.Logger(r.Context()))
	}
}
