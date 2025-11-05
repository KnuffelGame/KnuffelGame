package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetLobbyHandler returns an http.HandlerFunc that retrieves lobby details
// Headers required: X-User-ID, X-Username (from Gateway) OR AuthMiddleware must have injected user into context
// Path parameter: lobby_id (UUID)
// Returns lobby details with all players, marking the lobby leader
func GetLobbyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "get_lobby"))

		// Prefer user from context (set by AuthMiddleware). Fallback to headers to keep existing tests working.
		var (
			userID   uuid.UUID
			username string
		)
		if u, ok := auth.FromContext(r.Context()); ok {
			userID = u.ID
			username = u.Username
		} else {
			userIDStr := r.Header.Get(headerUserID)
			username = r.Header.Get(headerUsername)

			if userIDStr == "" || username == "" {
				log.Warn("missing required headers", slog.String("user_id", userIDStr), slog.String("username", username))
				httpx.WriteBadRequest(w, "Missing required headers: X-User-ID and X-Username", nil, log)
				return
			}

			var err error
			userID, err = uuid.Parse(userIDStr)
			if err != nil {
				log.Warn("invalid user_id format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
				httpx.WriteBadRequest(w, "Invalid user ID format", map[string]interface{}{"detail": err.Error()}, log)
				return
			}
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

		// Query lobby details with JOIN to get all players
		// JOIN: lobbies + players + users
		query := `
			SELECT 
				l.id as lobby_id,
				l.join_code,
				l.status,
				l.leader_id,
				p.id as player_id,
				p.user_id,
				u.username,
				p.joined_at,
				p.is_active
			FROM lobbies l
			LEFT JOIN players p ON l.id = p.lobby_id
			LEFT JOIN users u ON p.user_id = u.id
			WHERE l.id = $1
			ORDER BY p.joined_at ASC
		`

		rows, err := db.Query(query, lobbyID)
		if err != nil {
			log.Error("failed to query lobby", slog.String("error", err.Error()), slog.String("lobby_id", lobbyID.String()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}
		defer rows.Close()

		var response models.LobbyDetailResponse
		var players []models.PlayerInfo
		lobbyFound := false

		for rows.Next() {
			var (
				lobbyIDScanned uuid.UUID
				joinCode       string
				status         string
				leaderID       uuid.UUID
				playerID       uuid.NullUUID
				playerUserID   uuid.NullUUID
				playerUsername sql.NullString
				playerJoinedAt sql.NullTime
				playerIsActive sql.NullBool
			)

			err := rows.Scan(
				&lobbyIDScanned,
				&joinCode,
				&status,
				&leaderID,
				&playerID,
				&playerUserID,
				&playerUsername,
				&playerJoinedAt,
				&playerIsActive,
			)
			if err != nil {
				log.Error("failed to scan row", slog.String("error", err.Error()))
				httpx.WriteInternalError(w, "Database error", nil, log)
				return
			}

			// Set lobby details on first row
			if !lobbyFound {
				response.LobbyID = lobbyIDScanned
				response.JoinCode = joinCode
				response.Status = status
				response.LeaderID = leaderID
				lobbyFound = true
			}

			// Add player if present (LEFT JOIN might have NULLs if no players yet)
			if playerID.Valid && playerUserID.Valid {
				players = append(players, models.PlayerInfo{
					ID:       playerID.UUID,
					UserID:   playerUserID.UUID,
					Username: playerUsername.String,
					JoinedAt: playerJoinedAt.Time,
					IsActive: playerIsActive.Bool,
				})
			}
		}

		if err := rows.Err(); err != nil {
			log.Error("error iterating rows", slog.String("error", err.Error()))
			httpx.WriteInternalError(w, "Database error", nil, log)
			return
		}

		// Check if lobby was found
		if !lobbyFound {
			log.Info("lobby not found", slog.String("lobby_id", lobbyID.String()), slog.String("user_id", userID.String()))
			httpx.WriteNotFound(w, "Lobby not found", log)
			return
		}

		response.Players = players

		// Authorization: ensure requesting user is a member of the lobby (or the leader)
		isMember := false
		if response.LeaderID == userID {
			isMember = true
		} else {
			for _, p := range response.Players {
				if p.UserID == userID {
					isMember = true
					break
				}
			}
		}

		if !isMember {
			log.Warn("user is not a member of lobby", slog.String("lobby_id", lobbyID.String()), slog.String("user_id", userID.String()))
			httpx.WriteForbidden(w, "User is not a member of the lobby", log)
			return
		}

		log.Info("lobby details retrieved",
			slog.String("lobby_id", lobbyID.String()),
			slog.String("user_id", userID.String()),
			slog.Int("player_count", len(players)))

		httpx.WriteJSON(w, http.StatusOK, response, log)
	}
}
