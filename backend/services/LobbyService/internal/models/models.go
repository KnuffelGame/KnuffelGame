package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Lobby represents a game lobby
type Lobby struct {
	ID        uuid.UUID `json:"id" db:"id"`
	JoinCode  string    `json:"join_code" db:"join_code"`
	LeaderID  uuid.UUID `json:"leader_id" db:"leader_id"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Player represents a player in a lobby
type Player struct {
	ID       uuid.UUID  `json:"id" db:"id"`
	LobbyID  uuid.UUID  `json:"lobby_id" db:"lobby_id"`
	UserID   uuid.UUID  `json:"user_id" db:"user_id"`
	JoinedAt time.Time  `json:"joined_at" db:"joined_at"`
	IsActive bool       `json:"is_active" db:"is_active"`
	LeftAt   *time.Time `json:"left_at,omitempty" db:"left_at"`
}

// Lobby status constants
const (
	LobbyStatusWaiting  = "waiting"
	LobbyStatusInGame   = "running"
	LobbyStatusFinished = "finished"
)

// PlayerInfo represents a player in the response with user information
type PlayerInfo struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joined_at"`
	IsActive bool      `json:"is_active"`
}

// CreateLobbyResponse represents the response when creating a lobby
type CreateLobbyResponse struct {
	LobbyID  uuid.UUID    `json:"lobby_id"`
	JoinCode string       `json:"join_code"`
	LeaderID uuid.UUID    `json:"leader_id"`
	Status   string       `json:"status"`
	Players  []PlayerInfo `json:"players"`
}

// LobbyDetailResponse represents the response when getting lobby details
// Same structure as CreateLobbyResponse
type LobbyDetailResponse struct {
	LobbyID  uuid.UUID    `json:"lobby_id"`
	JoinCode string       `json:"join_code"`
	Status   string       `json:"status"`
	LeaderID uuid.UUID    `json:"leader_id"`
	Players  []PlayerInfo `json:"players"`
}

// JoinLobbyRequest represents the request to join a lobby by join code
type JoinLobbyRequest struct {
	JoinCode string `json:"join_code" validate:"required,len=6"`
}

// KickPlayerRequest represents the request to kick a player from a lobby
type KickPlayerRequest struct {
	TargetUserID string `json:"target_user_id" validate:"required,uuid"`
}

// UpdatePlayerActiveStatusRequest represents the request to update a player's active status
type UpdatePlayerActiveStatusRequest struct {
	IsActive bool `json:"is_active"`
}
