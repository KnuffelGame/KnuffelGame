package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
)

func (s *GameService) ToggleDice(ctx context.Context, gameID, userID string, indices []int) (*models.ToggleDiceResponse, error) {
	// A. Spiel laden
	game, err := s.Repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, ErrGameNotFound
	}

	// B. VALIDIERUNG: Ist der User dran?
	if err := s.validateTurn(game, userID); err != nil {
		log.Println("WARNING: validateTurn failed: ", err)
		return nil, err
	}

	// C. VALIDIERUNG: Roll Count > 0 und < 3
	currentTurn, err := s.Repo.GetCurrentTurn(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}
	if currentTurn == nil {
		return nil, fmt.Errorf("current turn not found")
	}

	if currentTurn.RollCount == 0 {
		return nil, fmt.Errorf("Cannot lock dice before first roll") // ErrInvalidRollCount
	}
	if currentTurn.RollCount >= 3 {
		return nil, fmt.Errorf("Cannot lock dice after 3rd roll") // ErrInvalidRollCount
	}

	// D. VALIDIERUNG & AKTION: Indices prüfen und toggeln
	for _, idx := range indices {
		// Prüfen: 0-4
		if idx < 0 || idx >= 5 { // Annahme: Es gibt immer 5 Würfel
			return nil, fmt.Errorf("Invalid dice index: must be between 0 and 4") // ErrInvalidDiceIndex
		}

		// Aktion: Toggeln (true -> false, false -> true)
		currentTurn.KeptDice[idx] = !currentTurn.KeptDice[idx]
	}

	// E. Timeout Timer resetten (auf 40s)
	//currentTurn.TurnTimerExpiresAt = time.Now().Add(40 * time.Second)

	// F. Speichern & Event senden
	if err := s.Repo.UpdateTurn(ctx, currentTurn); err != nil {
		return nil, err
	}

	var result *models.ToggleDiceResponse
	result = &models.ToggleDiceResponse{
		GameID: gameID,
		Dice:   s.BuildDices(currentTurn.DiceValues, currentTurn.KeptDice),
	}

	return result, nil
}

func (s *GameService) BuildDices(values []int, locked []bool) []models.Dice {
	dices := make([]models.Dice, len(values))
	for i := range values {
		dices[i] = models.Dice{
			Value:  values[i],
			Locked: locked[i],
		}
	}
	return dices
}

func (s *GameService) SelectScoreField(ctx context.Context, gameID, userID, fieldName string) (*models.SelectScoreFieldResponse, error) {
	game, err := s.Repo.GetGameByID(ctx, gameID)
	if err != nil {
		log.Println("WARNING: GetGameByID failed: ", err)
		return nil, ErrGameNotFound
	}

	// B. VALIDIERUNG: Ist der User dran?
	if err := s.validateTurn(game, userID); err != nil {
		log.Println("WARNING: validateTurn failed: ", err)
		return nil, err
	}

	// C. VALIDIERUNG: Roll Count > 0 und < 3
	currentTurn, err := s.Repo.GetCurrentTurn(ctx, gameID, userID)
	if err != nil {
		log.Println("WARNING: GetCurrentTurn failed: ", err)
		return nil, err
	}
	if currentTurn == nil {
		return nil, fmt.Errorf("current turn not found")
	}

	if currentTurn.RollCount == 0 {
		return nil, models.ErrInvalidRollcount // ErrInvalidRollCount
	}

	score, err := CalculateFieldScore(currentTurn.DiceValues, fieldName)
	log.Println("the score is:", score)
	if err != nil {
		return nil, err
	}

	currentRound := game.Round
	// D. Score in Scorecard speichern
	var scorecard = models.ScorecardDB{
		GameID:      gameID,
		UserID:      userID,
		FieldName:   fieldName,
		Value:       score,
		RoundFilled: &currentRound,
	}

	if err := s.Repo.UpdateScorecard(ctx, &scorecard); err != nil {
		log.Println("WARNING: UpdateScorecard failed: ", err)
		return nil, err
	}
	var result *models.SelectScoreFieldResponse
	result = &models.SelectScoreFieldResponse{
		GameID:    gameID,
		FieldName: fieldName,
		Score:     score,
	}

	// reset Turn for current player
	if err := s.Repo.ResetTurnAfterFinishedTurn(ctx, gameID, userID, currentRound); err != nil {
		log.Println("WARNING: ResetTurnAfterFinishedTurn failed: ", err)
		return nil, err
	}

	if err := s.SetNextPlayer(ctx, game); err != nil {
		log.Println("WARNING: SetNextPlayer failed: ", err)
		return nil, err
	}

	return result, nil
}

func (s *GameService) SetNextPlayer(ctx context.Context, game *models.GameDB) error {
	game.CurrentTurn++

	// Check if round is over
	if (game.CurrentTurn-1)%len(game.TurnOrder) == 0 {
		game.Round++
	}

	// Check if game is over (e.g. after 13 rounds)
	if game.Round > 13 {
		game.Status = "FINISHED"
		now := time.Now()
		game.EndedAt = &now
	}

	if err := s.Repo.UpdateGame(ctx, game); err != nil {
		return err
	}

	return nil
}
