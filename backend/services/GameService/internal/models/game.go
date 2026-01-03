package models

// Game represents the entire game, including its state and logic.
type Game struct {
	State GameState `json:"state"`
	// db connection would be here, but we'll pass it into methods for now.
}

// GameState represents the current state of the game.
// This struct is serialized to JSON and sent to the player frontends.
type GameState struct {
	GameID                  string       `json:"game_id"`
	LobbyID                 string       `json:"lobby_id"`
	Status                  string       `json:"status"`
	CurrentPlayerID         string       `json:"current_player_id"`
	CurrentPlayerUsername   string       `json:"current_player_username"`
	RollCount               int          `json:"roll_count"`
	Dice                    []Dice       `json:"dice"`
	TimeoutRemainingSeconds int          `json:"timeout_remaining_seconds"`
	TurnOrder               []Player     `json:"turn_order"`
	ScoreBoard              []ScoreBoard `json:"score_board"`
	Round                   int          `json:"round"`
}

// Dice repräsentiert einen einzelnen Würfel im Würfelbecher.
type Dice struct {
	Value  int  `json:"value"`
	Locked bool `json:"locked"`
}

// ScoreBoard repräsentiert die Punktetabelle eines einzelnen Spielers.
type ScoreBoard struct {
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

type Player struct {
	PlayerID string `json:"player_id"`
	Username string `json:"username"`
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
