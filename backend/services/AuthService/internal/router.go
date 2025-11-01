package router

import (
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/go-chi/chi/v5"
)

// New constructs the HTTP router using the provided JWT generator and validator and attaches logging middleware.
func New(gen *jwt.Generator, val *jwt.Validator) http.Handler {
	r := chi.NewRouter()
	// replace chi default logger with structured slog based middleware
	l := logger.Default()
	r.Use(logger.ChiMiddleware(l))

	// Healthcheck
	healthcheck.Mount(r)

	// Routes
	// POST /internal/create -> create guest user token
	r.Post("/internal/create", handlers.CreateTokenHandler(gen))
	// POST /internal/validate -> validate guest user token
	r.Post("/internal/validate", handlers.ValidateTokenHandler(val))

	return r
}
