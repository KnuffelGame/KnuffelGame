package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/google/uuid"
)

// Repository defines database operations required by the Lobby service.
type Repository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateUserIfNotExistsTx(tx *sql.Tx, userID uuid.UUID, username string) error
	CreateLobbyTx(tx *sql.Tx, joinCode string, leaderID uuid.UUID) (uuid.UUID, error)
	AddPlayerTx(tx *sql.Tx, lobbyID uuid.UUID, userID uuid.UUID) (uuid.UUID, time.Time, error)
	GetLobbyDetail(ctx context.Context, lobbyID uuid.UUID) (*models.LobbyDetailResponse, error)
	GetLobbyLeaderID(ctx context.Context, lobbyID uuid.UUID) (uuid.UUID, error)
	IsMember(ctx context.Context, lobbyID uuid.UUID, userID uuid.UUID) (bool, error)
	GetLobbyByJoinCode(ctx context.Context, joinCode string) (*models.Lobby, error)
	GetLobbyPlayerCount(ctx context.Context, lobbyID uuid.UUID) (int, error)

	// Transaction-based versions for join lobby functionality
	GetLobbyByJoinCodeTx(tx *sql.Tx, joinCode string) (*models.Lobby, error)
	GetLobbyPlayerCountTx(tx *sql.Tx, lobbyID uuid.UUID) (int, error)
	IsMemberTx(tx *sql.Tx, lobbyID uuid.UUID, userID uuid.UUID) (bool, error)

	// Kick player functionality
	KickPlayerTx(tx *sql.Tx, lobbyID uuid.UUID, targetUserID uuid.UUID) error
}
