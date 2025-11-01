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
	LobbyStatusInGame   = "in_game"
	LobbyStatusFinished = "finished"
	LobbyStatusClosed   = "closed"
)
