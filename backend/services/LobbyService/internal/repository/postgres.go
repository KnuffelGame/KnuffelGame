package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/google/uuid"
)

// PostgresRepository implements Repository using a *sql.DB
type PostgresRepository struct {
	DB *sql.DB
}

// New creates a new PostgresRepository
func New(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{DB: db}
}

func (r *PostgresRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.DB.BeginTx(ctx, nil)
}

func (r *PostgresRepository) CreateUserIfNotExistsTx(tx *sql.Tx, userID uuid.UUID, username string) error {
	_, err := tx.Exec(`
		INSERT INTO users (id, username)
		VALUES ($1, $2)
		ON CONFLICT (id) DO NOTHING
	`, userID, username)
	return err
}

func (r *PostgresRepository) CreateLobbyTx(tx *sql.Tx, joinCode string, leaderID uuid.UUID) (uuid.UUID, error) {
	var lobbyID uuid.UUID
	if err := tx.QueryRow(`
		INSERT INTO lobbies (join_code, leader_id, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`, joinCode, leaderID, models.LobbyStatusWaiting).Scan(&lobbyID); err != nil {
		return uuid.Nil, err
	}
	return lobbyID, nil
}

func (r *PostgresRepository) AddPlayerTx(tx *sql.Tx, lobbyID uuid.UUID, userID uuid.UUID) (uuid.UUID, time.Time, error) {
	var playerID uuid.UUID
	var joinedAt time.Time
	if err := tx.QueryRow(`
		INSERT INTO players (lobby_id, user_id, is_active)
		VALUES ($1, $2, true)
		RETURNING id, joined_at
	`, lobbyID, userID).Scan(&playerID, &joinedAt); err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return playerID, joinedAt, nil
}

func (r *PostgresRepository) GetLobbyDetail(ctx context.Context, lobbyID uuid.UUID) (*models.LobbyDetailResponse, error) {
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

	rows, err := r.DB.QueryContext(ctx, query, lobbyID)
	if err != nil {
		return nil, err
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
			playerID       sql.NullString
			playerUserID   sql.NullString
			playerUsername sql.NullString
			playerJoinedAt sql.NullTime
			playerIsActive sql.NullBool
		)

		if err := rows.Scan(
			&lobbyIDScanned,
			&joinCode,
			&status,
			&leaderID,
			&playerID,
			&playerUserID,
			&playerUsername,
			&playerJoinedAt,
			&playerIsActive,
		); err != nil {
			return nil, err
		}

		if !lobbyFound {
			response.LobbyID = lobbyIDScanned
			response.JoinCode = joinCode
			response.Status = status
			response.LeaderID = leaderID
			lobbyFound = true
		}

		// playerID and playerUserID are strings; check validity
		if playerID.Valid && playerUserID.Valid {
			pid, err := uuid.Parse(playerID.String)
			if err != nil {
				return nil, err
			}
			puid, err := uuid.Parse(playerUserID.String)
			if err != nil {
				return nil, err
			}

			players = append(players, models.PlayerInfo{
				ID:       pid,
				UserID:   puid,
				Username: playerUsername.String,
				JoinedAt: playerJoinedAt.Time,
				IsActive: playerIsActive.Bool,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !lobbyFound {
		return nil, sql.ErrNoRows
	}

	response.Players = players
	return &response, nil
}

func (r *PostgresRepository) GetLobbyLeaderID(ctx context.Context, lobbyID uuid.UUID) (uuid.UUID, error) {
	var leaderIDStr string
	err := r.DB.QueryRowContext(ctx, `SELECT leader_id::text FROM lobbies WHERE id = $1`, lobbyID).Scan(&leaderIDStr)
	if err != nil {
		return uuid.Nil, err
	}
	leaderID, err := uuid.Parse(leaderIDStr)
	if err != nil {
		return uuid.Nil, err
	}
	return leaderID, nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, lobbyID uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM players WHERE lobby_id = $1 AND user_id = $2)`, lobbyID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PostgresRepository) GetLobbyByJoinCode(ctx context.Context, joinCode string) (*models.Lobby, error) {
	var lobby models.Lobby
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, join_code, leader_id, status, created_at, updated_at
		FROM lobbies
		WHERE join_code = $1
	`, joinCode).Scan(
		&lobby.ID,
		&lobby.JoinCode,
		&lobby.LeaderID,
		&lobby.Status,
		&lobby.CreatedAt,
		&lobby.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &lobby, nil
}

func (r *PostgresRepository) GetLobbyPlayerCount(ctx context.Context, lobbyID uuid.UUID) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM players
		WHERE lobby_id = $1 AND is_active = true
	`, lobbyID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Transaction-based versions for join lobby functionality
func (r *PostgresRepository) GetLobbyByJoinCodeTx(tx *sql.Tx, joinCode string) (*models.Lobby, error) {
	var lobby models.Lobby
	err := tx.QueryRow(`
		SELECT id, join_code, leader_id, status, created_at, updated_at
		FROM lobbies
		WHERE join_code = $1
	`, joinCode).Scan(
		&lobby.ID,
		&lobby.JoinCode,
		&lobby.LeaderID,
		&lobby.Status,
		&lobby.CreatedAt,
		&lobby.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &lobby, nil
}

func (r *PostgresRepository) GetLobbyPlayerCountTx(tx *sql.Tx, lobbyID uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM players
		WHERE lobby_id = $1 AND is_active = true
	`, lobbyID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) IsMemberTx(tx *sql.Tx, lobbyID uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE lobby_id = $1 AND user_id = $2 AND is_active = true)`, lobbyID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PostgresRepository) KickPlayerTx(tx *sql.Tx, lobbyID uuid.UUID, targetUserID uuid.UUID) error {
	_, err := tx.Exec(`
		UPDATE players
		SET is_active = false, left_at = NOW()
		WHERE lobby_id = $1 AND user_id = $2 AND is_active = true
	`, lobbyID, targetUserID)
	return err
}
