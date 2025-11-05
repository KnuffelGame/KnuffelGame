package router

import (
	"database/sql"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/joincode"
	"github.com/go-chi/chi/v5"
)

// New constructs the HTTP router with database and join code generator dependencies
func New(db *sql.DB, codeGen *joincode.Generator) http.Handler {
	r := chi.NewRouter()
	// replace chi default logger with structured slog based middleware
	l := logger.Default()
	r.Use(logger.ChiMiddleware(l))

	// Healthcheck
	healthcheck.Mount(r)

	// Lobby endpoints grouped under auth middleware
	r.Route("/lobbies", func(r chi.Router) {
		// Authentication middleware (reads X-User-ID / X-Username and injects user into context)
		r.Use(auth.AuthMiddleware)

		// Create lobby (any authenticated user)
		r.Post("/", handlers.CreateLobbyHandler(db, codeGen))

		// Get lobby details - require membership
		r.With(handlers.RequireLobbyMember(db)).Get("/{lobby_id}", handlers.GetLobbyHandler(db))

		// Other lobby routes can use RequireLobbyMember or RequireLobbyLeader as appropriate
		// e.g. r.With(handlers.RequireLobbyLeader(db)).Post("/{lobby_id}/start", handlers.StartLobbyHandler(db))
	})

	return r
}
