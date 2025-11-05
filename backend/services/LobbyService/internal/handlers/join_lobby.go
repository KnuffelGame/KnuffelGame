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
	"github.com/google/uuid"
)

const (
	maxPlayers = 6
)

// JoinLobbyHandler returns an http.HandlerFunc that joins an existing lobby by join code
// Headers required: X-User-ID, X-Username (from Gateway)
// Request body: JoinLobbyRequest with join_code field
// Returns: LobbyDetailResponse on success, various error responses on failure
func JoinLobbyHandler(repo repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "join_lobby"))

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

		// Parse and validate request body
		var req models.JoinLobbyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("failed to decode request body", slog.String("error", err.Error()))
			httpx.WriteBadRequest(w, "Invalid request body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		// Validate join code format (6 characters, alphanumeric)
		if len(req.JoinCode) != 6 {
			log.Warn("invalid join code format", slog.String("join_code", req.JoinCode))
			httpx.WriteBadRequest(w, "Join code must be exactly 6 characters", nil, log)
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

		// 1. Find lobby by join code
		lobby, err := repo.GetLobbyByJoinCodeTx(tx, req.JoinCode)
		if err == sql.ErrNoRows {
			log.Info("lobby not found by join code", slog.String("join_code", req.JoinCode))
			httpx.WriteNotFound(w, "No lobby found with join code: "+req.JoinCode, log)
			return
		}
		if err != nil {
			log.Error("failed to find lobby by join code", slog.String("error", err.Error()), slog.String("join_code", req.JoinCode))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		// 2. Validate lobby status is "waiting"
		if lobby.Status != models.LobbyStatusWaiting {
			log.Info("lobby not joinable", slog.String("lobby_id", lobby.ID.String()), slog.String("status", lobby.Status))
			httpx.WriteError(w, http.StatusConflict, "lobby_not_joinable", "Cannot join lobby - game already started", nil, log)
			return
		}

		// 3. Check player count is less than max
		playerCount, err := repo.GetLobbyPlayerCountTx(tx, lobby.ID)
		if err != nil {
			log.Error("failed to get player count", slog.String("error", err.Error()), slog.String("lobby_id", lobby.ID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		if playerCount >= maxPlayers {
			log.Info("lobby is full", slog.String("lobby_id", lobby.ID.String()), slog.Int("player_count", playerCount))
			httpx.WriteError(w, http.StatusConflict, "lobby_full", "Lobby has reached maximum capacity (6 players)", nil, log)
			return
		}

		// 4. Check if user is already in lobby
		isMember, err := repo.IsMemberTx(tx, lobby.ID, userID)
		if err != nil {
			log.Error("failed to check membership", slog.String("error", err.Error()), slog.String("lobby_id", lobby.ID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		if isMember {
			log.Info("user already in lobby", slog.String("lobby_id", lobby.ID.String()), slog.String("user_id", userID.String()))
			httpx.WriteError(w, http.StatusConflict, "already_in_lobby", "You are already in this lobby", nil, log)
			return
		}

		// 5. Create user entry if not exists
		if err := repo.CreateUserIfNotExistsTx(tx, userID, username); err != nil {
			log.Error("failed to insert user", slog.String("error", err.Error()), slog.String("user_id", userID.String()))
			httpx.WriteInternalError(w, "Failed to create user", nil, log)
			return
		}

		// 6. Add user as player to lobby
		if _, _, err := repo.AddPlayerTx(tx, lobby.ID, userID); err != nil {
			log.Error("failed to add player to lobby", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Failed to add player to lobby", nil, log)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			log.Error("failed to commit transaction", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		// 7. Get updated lobby details to return
		lobbyDetail, err := repo.GetLobbyDetail(r.Context(), lobby.ID)
		if err != nil {
			log.Error("failed to get lobby details after joining", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Failed to get lobby details", nil, log)
			return
		}

		log.Info("user joined lobby successfully",
			slog.String("lobby_id", lobby.ID.String()),
			slog.String("join_code", req.JoinCode),
			slog.String("user_id", userID.String()),
			slog.String("username", username))

		httpx.WriteJSON(w, http.StatusOK, lobbyDetail, log)
	}
}
