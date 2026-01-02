package models

import "errors"

var (
	ErrGameNotFound          = errors.New("game_not_found")
	ErrNotYourTurn           = errors.New("not_your_turn")
	ErrMaxRolls              = errors.New("max_rolls_reached")
	ErrGameNotActive         = errors.New("game_not_active")
	ErrInternal              = errors.New("internal_error")
	ErrInvalidRollcount      = errors.New("invalid_dice_roll_count")
	ErrBonusCannotBeSelected = errors.New("bonus_cannot_be_selected")
)
