package models

import "time"

type GameState struct {
	GameID                  string            `json:"game_id"`
	LobbyID                 string            `json:"lobby_id"`
	Status                  string            `json:"status"`
	CurrentPlayerID         string            `json:"current_player_id"`
	CurrentPlayerUsername   string            `json:"current_player_username"`
	RollCount               int               `json:"roll_count"`
	Dice                    []Dice            `json:"dice"`
	TimeoutRemainingSeconds int               `json:"timeout_remaining_seconds"`
	TurnOrder               []string          `json:"turn_order"`
	ScoreBoard              []ScoreBoardEntry `json:"score_board"`
}

// Dice repräsentiert einen einzelnen Würfel im Würfelbecher.
type Dice struct {
	Value  int  `json:"value"`
	Locked bool `json:"locked"`
}

// ScoreBoardEntry repräsentiert die Punktetabelle eines einzelnen Spielers.
type ScoreBoardEntry struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Scores   Scores `json:"scores"`
}

type Scores struct {
	// Oberer Block
	Ones     *int `json:"ones"`
	Twos     *int `json:"twos"`
	Threes   *int `json:"threes"`
	Fours    *int `json:"fours"`
	Fives    *int `json:"fives"`
	Sixes    *int `json:"sixes"`
	UpperSum int  `json:"upper_sum"`
	Bonus    *int `json:"bonus"`

	// Unterer Block
	ThreeOfAKind  *int `json:"three_of_a_kind"`
	FourOfAKind   *int `json:"four_of_a_kind"`
	FullHouse     *int `json:"full_house"`
	SmallStraight *int `json:"small_straight"`
	LargeStraight *int `json:"large_straight"`
	Kniffel       *int `json:"kniffel"`
	Chance        *int `json:"chance"`
	LowerSum      int  `json:"lower_sum"`

	// Gesamt
	Total int `json:"total"`
}

type TurnOrderEntry struct {
	UserID   string `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// CreateGameRequest entspricht dem Schema '#/components/schemas/CreateGameRequest'
type CreateGameRequest struct {
	LobbyID   string           `json:"lobby_id" binding:"required"`
	TurnOrder []TurnOrderEntry `json:"turn_order" binding:"required,min=1"` // min=1 stellt sicher, dass wir mind. 1 Spieler haben
}

// CreateGameResponse entspricht dem Schema '#/components/schemas/CreateGameResponse'
type CreateGameResponse struct {
	GameID          string           `json:"game_id"`
	LobbyID         string           `json:"lobby_id"`
	CurrentPlayerID string           `json:"current_player_id"`
	TurnOrder       []TurnOrderEntry `json:"turn_order"`
}

// --- DB-STRUCTS (basierend auf deinem SQL-Schema) ---

// GameDB ist das Go-Struct, das die 'games'-Tabelle abbildet
type GameDB struct {
	ID          string     `json:"id"`
	LobbyID     string     `json:"lobby_id"`
	Status      string     `json:"status"` // 'active', 'finished' etc.
	CurrentTurn int        `json:"current_turn"`
	TurnOrder   []byte     `json:"turn_order"` // Wird als JSONB gespeichert
	Round       int        `json:"round"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"` // Zeiger, da es NULL sein kann
}

// ScorecardDB ist das Go-Struct, das die 'scorecards'-Tabelle abbildet
type ScorecardDB struct {
	ID          string `json:"id"`
	GameID      string `json:"game_id"`
	UserID      string `json:"user_id"`
	FieldName   string `json:"field_name"` // 'ones', 'twos' etc.
	Value       int    `json:"value"`
	RoundFilled *int   `json:"round_filled"` // Zeiger, da es NULL sein kann
}

// TurnDB ist das Go-Struct, das die 'turns'-Tabelle abbildet
type TurnDB struct {
	ID         string     `json:"id"`
	GameID     string     `json:"game_id"`
	UserID     string     `json:"user_id"`
	RollCount  int        `json:"roll_count"`
	DiceValues []int      `json:"dice_values"` // z.B. [1, 5, 2, 6, 1]
	KeptDice   []bool     `json:"kept_dice"`   // z.B. [false, true, false, false, true]
	Timeout    bool       `json:"timeout"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
}

// models/responses.go (oder in deine existierende Datei)

// RollDiceResponse entspricht dem Schema '#/components/schemas/RollDiceResponse'
type RollDiceResponse struct {
	GameID          string `json:"game_id"`
	RollCount       int    `json:"roll_count"`
	Dice            []Dice `json:"dice"`
	CanRollAgain    bool   `json:"can_roll_again"`
	MustSelectField bool   `json:"must_select_field"`
}

// ErrorResponse definiert die Struktur für detaillierte Fehler (403)
type ErrorResponse struct {
	Error   string       `json:"error"`   // z.B. "forbidden"
	Message string       `json:"message"` // z.B. "It's not your turn"
	Details ErrorDetails `json:"details,omitempty"`
}

type ErrorDetails struct {
	CurrentPlayer string `json:"current_player,omitempty"`
	RollCount     int    `json:"roll_count,omitempty"`
}
