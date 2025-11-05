package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RequireLobbyMember returns a middleware that ensures the requesting user is a member of the lobby (or the leader).
func RequireLobbyMember(repo repository.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(slog.String("action", "require_lobby_member"))

			// Get user from context (set by AuthMiddleware)
			user, ok := auth.FromContext(r.Context())
			if !ok {
				log.Warn("user missing from context")
				httpx.WriteUnauthorized(w, "Missing authentication headers", log)
				return
			}

			// Parse lobby_id from URL
			lobbyIDStr := chi.URLParam(r, "lobby_id")
			if lobbyIDStr == "" {
				log.Warn("missing lobby_id parameter")
				httpx.WriteBadRequest(w, "Missing lobby_id parameter", nil, log)
				return
			}

			lobbyID, err := uuid.Parse(lobbyIDStr)
			if err != nil {
				log.Warn("invalid lobby_id format", slog.String("lobby_id", lobbyIDStr), slog.String("error", err.Error()))
				httpx.WriteBadRequest(w, "Invalid lobby ID format", map[string]interface{}{"detail": err.Error()}, log)
				return
			}

			// First check if user is the leader
			leaderID, err := repo.GetLobbyLeaderID(r.Context(), lobbyID)
			if err == sql.ErrNoRows {
				log.Info("lobby not found", slog.String("lobby_id", lobbyIDStr))
				httpx.WriteNotFound(w, "Lobby not found", log)
				return
			}
			if err != nil {
				log.Error("failed to query lobby leader", slog.String("error", err.Error()))
				httpx.WriteInternalError(w, "Database error", nil, log)
				return
			}

			if leaderID == user.ID {
				next.ServeHTTP(w, r)
				return
			}

			// Check membership
			exists, err := repo.IsMember(r.Context(), lobbyID, user.ID)
			if err != nil {
				log.Error("failed to query membership", slog.String("error", err.Error()))
				httpx.WriteInternalError(w, "Database error", nil, log)
				return
			}

			if !exists {
				log.Warn("user is not a member of lobby", slog.String("lobby_id", lobbyIDStr), slog.String("user_id", user.ID.String()))
				httpx.WriteForbidden(w, "User is not a member of the lobby", log)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireLobbyLeader returns a middleware that ensures the requesting user is the lobby leader.
func RequireLobbyLeader(repo repository.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(slog.String("action", "require_lobby_leader"))

			user, ok := auth.FromContext(r.Context())
			if !ok {
				log.Warn("user missing from context")
				httpx.WriteUnauthorized(w, "Missing authentication headers", log)
				return
			}

			lobbyIDStr := chi.URLParam(r, "lobby_id")
			if lobbyIDStr == "" {
				log.Warn("missing lobby_id parameter")
				httpx.WriteBadRequest(w, "Missing lobby_id parameter", nil, log)
				return
			}

			lobbyID, err := uuid.Parse(lobbyIDStr)
			if err != nil {
				log.Warn("invalid lobby_id format", slog.String("lobby_id", lobbyIDStr), slog.String("error", err.Error()))
				httpx.WriteBadRequest(w, "Invalid lobby ID format", map[string]interface{}{"detail": err.Error()}, log)
				return
			}

			leaderID, err := repo.GetLobbyLeaderID(r.Context(), lobbyID)
			if err == sql.ErrNoRows {
				log.Info("lobby not found", slog.String("lobby_id", lobbyIDStr))
				httpx.WriteNotFound(w, "Lobby not found", log)
				return
			}
			if err != nil {
				log.Error("failed to query lobby leader", slog.String("error", err.Error()))
				httpx.WriteInternalError(w, "Database error", nil, log)
				return
			}

			if leaderID != user.ID {
				log.Warn("user is not the leader", slog.String("lobby_id", lobbyIDStr), slog.String("user_id", user.ID.String()))
				httpx.WriteForbidden(w, "User is not the lobby leader", log)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
