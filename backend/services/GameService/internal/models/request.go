package models

type CreateGameRequest struct {
	LobbyID   string   `json:"lobby_id" binding:"required"`
	TurnOrder []Player `json:"turn_order" binding:"required,min=1"` // min=1 stellt sicher, dass wir mind. 1 Spieler haben
}

type ToggleDiceRequest struct {
	DiceIndices []int `json:"dice_indices"`
}

type SelectScoreFieldRequest struct {
	FieldName string `json:"field_name" binding:"required"`
}
