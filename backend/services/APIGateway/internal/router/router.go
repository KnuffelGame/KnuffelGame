package router

import (
	"net/http"

	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/internal/handlers"
	authmw "github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/internal/middleware"
	"github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/pkg/config"
)

// New creates and configures the HTTP router with middleware.
// Returns an http.Handler ready for use with an HTTP server.
func New(cfg *config.Config, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Request correlation and logging (outermost)
	r.Use(middleware.RequestID)
	r.Use(logger.ChiMiddleware(log))

	// Healthcheck (bypass auth)
	healthcheck.Mount(r)

	// Authenticated routes
	r.Route("/lobbies", func(r chi.Router) {
		r.Use(authmw.Authentication(cfg))
		r.Use(authmw.HeaderInjection)
		r.Handle("/*", handlers.ReverseProxy(cfg.LobbyServiceURL))
	})
	r.Route("/games", func(r chi.Router) {
		r.Use(authmw.Authentication(cfg))
		r.Use(authmw.HeaderInjection)
		r.Handle("/*", handlers.ReverseProxy(cfg.GameServiceURL))
	})

	// Ensure /internal/* is not publicly accessible
	r.Handle("/internal/*", http.NotFoundHandler())

	return r
}
