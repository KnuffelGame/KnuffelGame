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

// KickPlayerHandler returns an http.HandlerFunc that kicks a player from a lobby
// Headers required: X-User-ID, X-Username (from Gateway)
// Path parameter: lobby_id (UUID)
// Request body: KickPlayerRequest with target_user_id field
// Returns: 204 No Content on success, various error responses on failure
func KickPlayerHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "kick_player"))

		// Extract user context from headers
		userIDStr := r.Header.Get(headerUserID)
		username := r.Header.Get(headerUsername)

		if userIDStr == "" || username == "" {
			log.Warn("missing required headers", slog.String("user_id", userIDStr), slog.String("username", username))
			httpx.WriteBadRequest(w, "Missing required headers: X-User-ID and X-Username", nil, log)
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Warn("invalid user_id format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid user ID format", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

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

		// Parse and validate request body
		var req models.KickPlayerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("failed to decode request body", slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid request body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		targetUserID, err := uuid.Parse(req.TargetUserID)
		if err != nil {
			log.Warn("invalid target_user_id format", slog.String("target_user_id", req.TargetUserID), slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid target user ID format", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		// Cannot kick yourself
		if targetUserID == userID {
			log.Warn("cannot kick yourself", slog.String("user_id", userID.String()))
			httpx.WriteError(w, http.StatusBadRequest, "cannot_kick_self", "Cannot kick yourself", nil, log)
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

		// 1. Check that the requesting user is the lobby leader
		leaderID, err := repo.GetLobbyLeaderID(r.Context(), lobbyID)
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

		if leaderID != userID {
			log.Warn("user is not lobby leader", slog.String("lobby_id", lobbyID.String()), slog.String("user_id", userID.String()), slog.String("leader_id", leaderID.String()))
			httpx.WriteForbidden(w, "Only the lobby leader can kick players", log)
			return
		}

		// 2. Check that target_user_id is in the lobby
		isMember, err := repo.IsMemberTx(tx, lobbyID, targetUserID)
		if err != nil {
			log.Error("failed to check membership", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()), slog.String("target_user_id", targetUserID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		if !isMember {
			log.Warn("target user is not in lobby", slog.String("lobby_id", lobbyID.String()), slog.String("target_user_id", targetUserID.String()))
			httpx.WriteError(w, http.StatusNotFound, "player_not_in_lobby", "Target user is not in the lobby", nil, log)
			return
		}

		// 3. Update the player record: SET is_active=false, left_at=NOW()
		if err := repo.KickPlayerTx(tx, lobbyID, targetUserID); err != nil {
			log.Error("failed to kick player", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()), slog.String("target_user_id", targetUserID.String()))
			httpx.WriteInternalError(w, "Failed to kick player", nil, log)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Error("failed to commit transaction", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		log.Info("player kicked successfully",
			slog.String("lobby_id", lobbyID.String()),
			slog.String("user_id", userID.String()),
			slog.String("target_user_id", targetUserID.String()))

		httpx.WriteNoContent(w)
	}
}
