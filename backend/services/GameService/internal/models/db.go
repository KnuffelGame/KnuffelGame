package models

import "time"

type GameDB struct {
	ID          string     `json:"id"`
	LobbyID     string     `json:"lobby_id"`
	Status      string     `json:"status"`
	CurrentTurn int        `json:"current_turn"`
	TurnOrder   []Player   `json:"turn_order"`
	Round       int        `json:"round"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
}

type ScorecardDB struct {
	ID          string `json:"id"`
	GameID      string `json:"game_id"`
	UserID      string `json:"user_id"`
	FieldName   string `json:"field_name"`
	Value       int    `json:"value"`
	RoundFilled *int   `json:"round_filled"`
}

type TurnDB struct {
	ID         string     `json:"id"`
	GameID     string     `json:"game_id"`
	UserID     string     `json:"user_id"`
	RollCount  int        `json:"roll_count"`
	DiceValues []int      `json:"dice_values"`
	KeptDice   []bool     `json:"kept_dice"`
	Timeout    bool       `json:"timeout"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Round      int        `json:"round"`
}
