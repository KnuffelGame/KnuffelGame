package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
)

// New constructs the HTTP router using the provided JWT generator.
func New(gen *jwt.Generator) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Routes
	// POST /internal/create -> create guest user token
	r.Post("/internal/create", handlers.CreateTokenHandler(gen))

	return r
}
