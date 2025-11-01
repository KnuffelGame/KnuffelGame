package router

import (
	"database/sql"
	"net/http"

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

	// Lobby endpoints
	r.Post("/lobbies", handlers.CreateLobbyHandler(db, codeGen))

	return r
}
