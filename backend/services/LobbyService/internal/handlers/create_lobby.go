package handlers

import (
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/joincode"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/google/uuid"
)

const (
	headerUserID   = "X-User-ID"
	headerUsername = "X-Username"
)

// CreateLobbyHandler returns an http.HandlerFunc that creates a new lobby
// Headers required: X-User-ID, X-Username (from Gateway)
// Creates user if not exists, generates join code, sets user as leader, adds user as first player
func CreateLobbyHandler(repo repository.Repository, codeGen *joincode.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "create_lobby"))

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

		// Begin transaction via repository
		tx, err := repo.BeginTx(r.Context())
		if err != nil {
			log.Error("failed to begin transaction", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}
		defer tx.Rollback()

		// 1. Create or get user (ON CONFLICT DO NOTHING)
		if err := repo.CreateUserIfNotExistsTx(tx, userID, username); err != nil {
			log.Error("failed to insert user", slog.String("error", err.Error()), slog.String("user_id", userID.String()))
			httpx.WriteInternalError(w, "Failed to create user", nil, log)
			return
		}

		// 2. Generate unique join code
		joinCode, err := codeGen.GenerateJoinCode()
		if err != nil {
			log.Error("failed to generate join code", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Failed to generate join code", nil, log)
			return
		}

		// 3. Create lobby with user as leader
		lobbyID, err := repo.CreateLobbyTx(tx, joinCode, userID)
		if err != nil {
			log.Error("failed to create lobby", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Failed to create lobby", nil, log)
			return
		}

		// 4. Add user as first player in the lobby
		playerID, joinedAt, err := repo.AddPlayerTx(tx, lobbyID, userID)
		if err != nil {
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

		// 5. Build response
		response := models.CreateLobbyResponse{
			LobbyID:  lobbyID,
			JoinCode: joinCode,
			LeaderID: userID,
			Status:   models.LobbyStatusWaiting,
			Players: []models.PlayerInfo{
				{
					ID:       playerID,
					UserID:   userID,
					Username: username,
					JoinedAt: joinedAt,
					IsActive: true,
				},
			},
		}

		log.Info("lobby created successfully",
			slog.String("lobby_id", lobbyID.String()),
			slog.String("join_code", joinCode),
			slog.String("leader_id", userID.String()))

		httpx.WriteJSON(w, http.StatusCreated, response, log)
	}
}
