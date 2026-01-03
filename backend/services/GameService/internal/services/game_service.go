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
		return nil, models.ErrGameNotFound
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
		return nil, models.ErrGameNotFound
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

	// Validierung ist das Feld schon belegt
	var fieldAvailable bool
	fieldAvailable, err = s.Repo.CheckIfFieldIsAvailable(ctx, gameID, userID, fieldName)
	if err != nil {
		log.Println("WARNING: CheckIfFieldIsAvailable failed: ", err)
		return nil, err
	}

	if !fieldAvailable {
		return nil, models.ErrFieldAlreadySelected
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
		// TODO add method to calculate result
	}

	if err := s.Repo.UpdateGame(ctx, game); err != nil {
		return err
	}

	return nil
}

func (s *GameService) FinishGamed(ctx context.Context, gameID string) error {
	game, err := s.Repo.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}

	game.Status = "FINISHED"

	if err := s.Repo.UpdateGame(ctx, game); err != nil {
		return err
	}

	return nil
}

func (s *GameService) BuildGameState(ctx context.Context, gameID string) (*models.GameState, error) {
	game, err := s.Repo.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, models.ErrGameNotFound
	}

	currentTurn, err := s.Repo.GetCurrentTurn(ctx, gameID, game.TurnOrder[(game.CurrentTurn-1)%len(game.TurnOrder)].PlayerID)
	if err != nil || currentTurn == nil {
		log.Println("ERROR: GetCurrentTurn failed: ", err)
		return nil, err
	}

	var Scoreboards, _ = s.Repo.GetScoreBoard(ctx, gameID)

	var gameState = models.GameState{
		GameID:                gameID,
		LobbyID:               game.LobbyID,
		Status:                game.Status,
		CurrentPlayerID:       game.TurnOrder[(game.CurrentTurn-1)%len(game.TurnOrder)].PlayerID,
		CurrentPlayerUsername: game.TurnOrder[(game.CurrentTurn-1)%len(game.TurnOrder)].Username,
		RollCount:             currentTurn.RollCount,
		Dice:                  s.BuildDices(currentTurn.DiceValues, currentTurn.KeptDice),
		TurnOrder:             game.TurnOrder,
		ScoreBoard:            Scoreboards,
		Round:                 game.Round,
	}

	return &gameState, nil

}
