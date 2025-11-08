package router

import (
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/joincode"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
)

// New constructs the HTTP router with repository and join code generator dependencies
func New(repo repository.Repository, codeGen *joincode.Generator) http.Handler {
	r := chi.NewRouter()
	// replace chi default logger with structured slog based middleware
	l := logger.Default()
	r.Use(logger.ChiMiddleware(l))

	// Healthcheck
	healthcheck.Mount(r)

	// Internal endpoints (no auth required)
	r.Route("/internal", func(r chi.Router) {
		r.Route("/lobbies", func(r chi.Router) {
			r.Put("/{lobby_id}/players/{player_id}/active", handlers.UpdatePlayerActiveStatusHandler(repo))
		})
	})

	// Lobby endpoints grouped under auth middleware
	r.Route("/lobbies", func(r chi.Router) {
		// Authentication middleware (reads X-User-ID / X-Username and injects user into context)
		r.Use(auth.AuthMiddleware)

		// Create lobby (any authenticated user)
		r.Post("/", handlers.CreateLobbyHandler(repo, codeGen))

		// Join lobby (any authenticated user)
		r.Post("/join", handlers.JoinLobbyHandler(repo))

		// Get lobby details - require membership
		r.With(handlers.RequireLobbyMember(repo)).Get("/{lobby_id}", handlers.GetLobbyHandler(repo))

		// Kick player - require leadership
		r.With(handlers.RequireLobbyLeader(repo)).Post("/{lobby_id}/kick", handlers.KickPlayerHandler(repo))

		// Other lobby routes can use RequireLobbyMember or RequireLobbyLeader as appropriate
		// e.g. r.With(handlers.RequireLobbyLeader(repo)).Post("/{lobby_id}/start", handlers.StartLobbyHandler(db))
	})

	return r
}
