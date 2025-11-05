package models

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
