package database

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrGameNotFound = errors.New("game not found")

type Repository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewRepository(db *pgxpool.Pool, logger *slog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

func (r *Repository) GetGameStateByID(ctx context.Context, gameID string) (*models.GameState, error) {
	query := "SELECT state FROM games WHERE id = $1"

	var jsonState []byte

	err := r.db.QueryRow(ctx, query, gameID).Scan(&jsonState)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("Game not found in DB", "game_id", gameID)
			return nil, ErrGameNotFound
		}
		// Fall 2: Anderer Datenbankfehler
		r.logger.Error("Database query failed", "error", err, "game_id", gameID)
		return nil, err
	}

	var gameState models.GameState
	if err := json.Unmarshal(jsonState, &gameState); err != nil {
		r.logger.Error("Database unmarshal failed", "error", err, "game_id", gameID)
	}
	return &gameState, nil
}
