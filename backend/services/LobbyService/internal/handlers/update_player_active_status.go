package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UpdatePlayerActiveStatusHandler returns an http.HandlerFunc that updates a player's active status in a lobby
// Path parameters: lobby_id (UUID), player_id (UUID)
// Request body: UpdatePlayerActiveStatusRequest with is_active boolean field
// Returns: 204 No Content on success, various error responses on failure
func UpdatePlayerActiveStatusHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "update_player_active_status"))

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

		// Extract player_id from URL path parameter
		playerIDStr := chi.URLParam(r, "player_id")
		if playerIDStr == "" {
			log.Warn("missing player_id parameter")
			httpx.WriteBadRequest(w, "Missing player_id parameter", nil, log)
			return
		}

		playerID, err := uuid.Parse(playerIDStr)
		if err != nil {
			log.Warn("invalid player_id format", slog.String("player_id", playerIDStr), slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid player ID format", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		// Parse and validate request body
		var req models.UpdatePlayerActiveStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("failed to decode request body", slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid request body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		// Begin transaction via repository
		tx, err := repo.BeginTx(r.Context())
		if err != nil {
			log.Error("failed to begin transaction", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}
		defer tx.Rollback()

		// 1. Validate that the lobby exists
		_, err = repo.GetLobbyLeaderID(r.Context(), lobbyID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Warn("lobby not found", slog.String("lobby_id", lobbyID.String()))
				httpx.WriteNotFound(w, "Lobby not found", log)
				return
			}
			log.Error("failed to get lobby leader", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		// 2. Validate that the player exists in the lobby
		playerExists, err := repo.IsMemberTx(tx, lobbyID, playerID)
		if err != nil {
			log.Error("failed to check player membership", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()), slog.String("player_id", playerID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		if !playerExists {
			log.Warn("player not found in lobby", slog.String("lobby_id", lobbyID.String()), slog.String("player_id", playerID.String()))
			httpx.WriteNotFound(w, "Player not found in lobby", log)
			return
		}

		// 3. Update the player's active status using the repository method
		if err := repo.UpdatePlayerActiveStatusTx(tx, lobbyID, playerID, req.IsActive); err != nil {
			if err == sql.ErrNoRows {
				log.Warn("player not found for update", slog.String("lobby_id", lobbyID.String()), slog.String("player_id", playerID.String()))
				httpx.WriteNotFound(w, "Player not found", log)
				return
			}
			log.Error("failed to update player active status", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()), slog.String("player_id", playerID.String()))
			httpx.WriteInternalError(w, "Failed to update player active status", nil, log)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Error("failed to commit transaction", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		log.Info("player active status updated successfully",
			slog.String("lobby_id", lobbyID.String()),
			slog.String("player_id", playerID.String()),
			slog.Bool("is_active", req.IsActive))

		httpx.WriteNoContent(w)
	}
}
