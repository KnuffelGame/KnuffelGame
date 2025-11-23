package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
)

var ErrGameNotFound = errors.New("game not found")

// DBTX definiert die Methoden, die sql.DB und sql.Tx gemeinsam haben.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Repository struct {
	db DBTX
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// WithTransaction startet eine Transaktion.
func (r *Repository) WithTransaction(ctx context.Context, fn func(*Repository) error) error {
	// 1. Prüfen: Sind wir überhaupt auf der Hauptverbindung (*sql.DB)?
	// Wenn r.db bereits eine Transaktion (*sql.Tx) ist, können wir keine neue starten
	// (außer wir nutzen Savepoints, was hier zu weit führt).
	db, ok := r.db.(*sql.DB)
	if !ok {
		return fmt.Errorf("current repository is already in a transaction")
	}

	// 2. Transaktion starten
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// 3. Temporäres Repo erstellen, das die Transaktion nutzt
	txRepo := &Repository{db: tx}

	// 4. Logik ausführen
	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	// 5. Commit
	return tx.Commit()
}

func (r *Repository) CreateGame(ctx context.Context, game *models.GameDB) error {
	query := `
        INSERT INTO games (id, lobby_id, status, current_turn, turn_order, round, started_at, ended_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	// Hinweis: game.TurnOrder ist bereits []byte oder string (vom Handler vorbereitet),
	// daher können wir es direkt übergeben.
	_, err := r.db.ExecContext(ctx, query,
		game.ID,
		game.LobbyID,
		game.Status,
		game.CurrentTurn,
		game.TurnOrder,
		game.Round,
		game.StartedAt,
		game.EndedAt, // Kann nil sein, sql.DB behandelt das korrekt als NULL
	)

	if err != nil {
		return fmt.Errorf("failed to insert game: %w", err)
	}
	return nil
}

func (r *Repository) CreateTurn(ctx context.Context, m *models.TurnDB) error {
	query := `INSERT INTO turns (id, game_id, user_id, roll_count, dice_values, kept_dice, timeout, started_at, ended_at)
        	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// 1. DiceValues ([]int) zu JSON konvertieren
	diceJSON, err := json.Marshal(m.DiceValues)
	if err != nil {
		return fmt.Errorf("failed to marshal dice_values: %w", err)
	}

	// 2. KeptDice ([]bool) zu JSON konvertieren
	keptJSON, err := json.Marshal(m.KeptDice)
	if err != nil {
		return fmt.Errorf("failed to marshal kept_dice: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		m.ID,
		m.GameID,
		m.UserID,
		m.RollCount,
		diceJSON, // Übergabe als JSON bytes
		keptJSON, // Übergabe als JSON bytes
		m.Timeout,
		m.StartedAt,
		m.EndedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert turn: %w", err)
	}
	return nil
}

func (r *Repository) CreateScorecard(ctx context.Context, m *models.ScorecardDB) error {
	query := `
        INSERT INTO scorecards (id, game_id, user_id, field_name, value, round_filled)
        VALUES ($1, $2, $3, $4, $5, $6)
    `

	_, err := r.db.ExecContext(ctx, query,
		m.ID,
		m.GameID,
		m.UserID,
		m.FieldName,
		m.Value,
		m.RoundFilled, // Ist ein Pointer (*int). Wenn nil -> SQL NULL.
	)

	if err != nil {
		return fmt.Errorf("failed to insert scorecard: %w", err)
	}
	return nil
}
