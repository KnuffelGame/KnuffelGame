package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetLobbyInternalHandler returns an http.HandlerFunc that retrieves lobby details for internal use
// Path parameter: lobby_id (UUID)
// Returns lobby details with all players, no authentication or authorization required
func GetLobbyInternalHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "get_lobby_internal"))

		// Extract lobby_id from URL path parameter
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

		// Query lobby via repository
		response, err := repo.GetLobbyDetail(r.Context(), lobbyID)
		if err == sql.ErrNoRows {
			log.Info("lobby not found", slog.String("lobby_id", lobbyID.String()))
			httpx.WriteNotFound(w, "Lobby not found", log)
			return
		}
		if err != nil {
			log.Error("failed to query lobby", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		log.Info("lobby details retrieved",
			slog.String("lobby_id", lobbyID.String()),
			slog.Int("player_count", len(response.Players)))

		httpx.WriteJSON(w, http.StatusOK, response, log)
	}
}
