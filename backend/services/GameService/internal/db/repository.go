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
	query := `INSERT INTO turns (id, game_id, user_id, roll_count, dice_values, kept_dice, timeout, started_at, ended_at, round)
        	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

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
		m.Round,
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

func (r *Repository) GetGameByID(ctx context.Context, gameID string) (*models.GameDB, error) {
	query := `SELECT id, lobby_id, status, current_turn, turn_order, round, started_at, ended_at 
              FROM games WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, gameID)

	var g models.GameDB
	var turnOrderJSON []byte // 1. Temporäre Variable für die rohen JSON-Bytes

	// 2. Scan: turnOrderJSON anstelle von &g.TurnOrder nutzen
	err := row.Scan(
		&g.ID,
		&g.LobbyID,
		&g.Status,
		&g.CurrentTurn,
		&turnOrderJSON, // Hier kommen die Bytes rein
		&g.Round,
		&g.StartedAt,
		&g.EndedAt,
	)
	if err != nil {
		return nil, err
	}

	// 3. Konvertierung: Bytes -> []string
	if len(turnOrderJSON) > 0 {
		if err := json.Unmarshal(turnOrderJSON, &g.TurnOrder); err != nil {
			return nil, fmt.Errorf("fehler beim parsen von turn_order: %w", err)
		}
	}

	return &g, nil
}

// UpdateTurn
func (r *Repository) UpdateTurn(ctx context.Context, t *models.TurnDB) error {
	diceValJSON, _ := json.Marshal(t.DiceValues)
	keptDiceJSON, _ := json.Marshal(t.KeptDice)

	query := `UPDATE turns 
              SET roll_count = $1, dice_values = $2, kept_dice = $3, started_at = $4 
              WHERE id = $5`

	// Auch ExecContext ist Teil von DBTX
	_, err := r.db.ExecContext(ctx, query, t.RollCount, diceValJSON, keptDiceJSON, t.StartedAt, t.ID)
	return err
}

// GetCurrentTurn sucht den aktuellen, noch offenen Spielzug des Users.
func (r *Repository) GetCurrentTurn(ctx context.Context, gameID, userID string) (*models.TurnDB, error) {
	// Die Query sucht nach einem Turn für diesen User in diesem Spiel.
	// WICHTIG: "AND ended_at IS NULL" stellt sicher, dass wir keinen alten, abgeschlossenen Zug laden.
	query := `
        SELECT id, game_id, user_id, roll_count, dice_values, kept_dice, timeout, started_at, ended_at
        FROM turns
        WHERE game_id = $1 AND user_id = $2 AND ended_at IS NULL
        LIMIT 1
    `

	var t models.TurnDB

	// Wir brauchen temporäre Variablen für die JSON-Spalten,
	// da SQL-Treiber JSON meist als []byte zurückgeben.
	var diceValJSON []byte
	var keptDiceJSON []byte

	// Ausführung über das DBTX Interface (r.db)
	err := r.db.QueryRowContext(ctx, query, gameID, userID).Scan(
		&t.ID,
		&t.GameID,
		&t.UserID,
		&t.RollCount,
		&diceValJSON,  // Scannt JSON-Daten
		&keptDiceJSON, // Scannt JSON-Daten
		&t.Timeout,
		&t.StartedAt,
		&t.EndedAt, // Pointer *time.Time fängt NULL sauber ab
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Das ist kein Fehler! Es bedeutet nur, dass der Spieler in dieser Runde
			// noch nicht gewürfelt hat. Wir geben nil zurück.
			return nil, nil
		}
		return nil, err
	}

	// JSON-Daten aus der DB in die Go-Structs (Slices) umwandeln
	// Falls die Felder in der DB NULL wären, sind die Byte-Slices leer/nil -> Unmarshal ignoriert das oder gibt Fehler.
	// Wir initialisieren sicherheitshalber leere Slices bei Fehler/Nil.

	if len(diceValJSON) > 0 {
		if err := json.Unmarshal(diceValJSON, &t.DiceValues); err != nil {
			return nil, err // oder loggen und leeren Slice lassen
		}
	} else {
		t.DiceValues = []int{}
	}

	if len(keptDiceJSON) > 0 {
		if err := json.Unmarshal(keptDiceJSON, &t.KeptDice); err != nil {
			return nil, err
		}
	} else {
		t.KeptDice = []bool{}
	}

	return &t, nil
}

func (r *Repository) UpdateScorecard(ctx context.Context, s *models.ScorecardDB) error {
	query := `UPDATE scorecards 
			  SET value = $1, round_filled = $2 
			  WHERE game_id = $3 AND user_id = $4 AND field_name = $5`

	_, err := r.db.ExecContext(ctx, query, s.Value, s.RoundFilled, s.GameID, s.UserID, s.FieldName)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateGame(ctx context.Context, g *models.GameDB) error {
	query := `UPDATE games 
			  SET current_turn = $1, round = $2, status = $3, ended_at = $4 
			  WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, g.CurrentTurn, g.Round, g.Status, g.EndedAt, g.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) ResetTurnAfterFinishedTurn(ctx context.Context, gameID, userID string, round int) error {
	keptDice := []bool{false, false, false, false, false}
	keptDiceJSON, err := json.Marshal(keptDice)
	if err != nil {
		return fmt.Errorf("failed to marshal kept_dice for reset: %w", err)
	}

	query := `UPDATE turns
			  SET roll_count = 0, kept_dice = $1, round = $2
			  WHERE game_id = $3 AND user_id = $4`

	_, err = r.db.ExecContext(ctx, query, keptDiceJSON, round, gameID, userID)
	return err
}
