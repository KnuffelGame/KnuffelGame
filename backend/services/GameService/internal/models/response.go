package models

type CreateGameResponse struct {
	GameID          string   `json:"game_id"`
	LobbyID         string   `json:"lobby_id"`
	CurrentPlayerID string   `json:"current_player_id"`
	TurnOrder       []Player `json:"turn_order"`
}

type RollDiceResponse struct {
	GameID          string `json:"game_id"`
	RollCount       int    `json:"roll_count"`
	Dice            []Dice `json:"dice"`
	CanRollAgain    bool   `json:"can_roll_again"`
	MustSelectField bool   `json:"must_select_field"`
}

type ToggleDiceResponse struct {
	GameID string `json:"game_id"`
	Dice   []Dice `json:"dice"`
}

type SelectScoreFieldResponse struct {
	GameID    string `json:"game_id"`
	FieldName string `json:"field_name"`
	Score     int    `json:"score"`
}
