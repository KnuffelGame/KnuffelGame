package services

import (
	"context"
	"fmt"
	"time"
)

func (s *GameService) ToggleDice(ctx context.Context, gameID, userID string, indices []int) (*Game, error) {
	// A. Spiel laden
	game, err := s.Repo.GetGame(gameID)
	if err != nil {
		return nil, ErrGameNotFound
	}

	// B. VALIDIERUNG: Ist der User dran?
	// Hier nutzen wir die Logik von vorhin (Modulo), um den Index sicher zu bestimmen
	playerIndex := game.CurrentTurn % len(game.TurnOrder)
	expectedPlayerID := game.TurnOrder[playerIndex].UserID

	if expectedPlayerID != userID {
		return nil, ErrNotYourTurn
	}

	// C. VALIDIERUNG: Roll Count > 0 und < 3
	// Man darf nur locken, NACHDEM man gewürfelt hat (Count 1) oder vor dem 3. Wurf (Count 2).
	// Wenn RollCount 0 ist: Noch nicht gewürfelt -> Fehler.
	// Wenn RollCount 3 ist: Zug ist "vorbei" (nur noch eintragen) -> Fehler.
	if game.RollCount == 0 {
		return nil, fmt.Errorf("Cannot lock dice before first roll") // ErrInvalidRollCount
	}
	if game.RollCount >= 3 {
		return nil, fmt.Errorf("Cannot lock dice after 3rd roll") // ErrInvalidRollCount
	}

	// D. VALIDIERUNG & AKTION: Indices prüfen und toggeln
	for _, idx := range indices {
		// Prüfen: 0-4
		if idx < 0 || idx >= 5 { // Annahme: Es gibt immer 5 Würfel
			return nil, fmt.Errorf("Invalid dice index: must be between 0 and 4") // ErrInvalidDiceIndex
		}

		// Aktion: Toggeln (true -> false, false -> true)
		game.Dice[idx].Locked = !game.Dice[idx].Locked
	}

	// E. Timeout Timer resetten (auf 40s)
	game.TurnTimerExpiresAt = time.Now().Add(40 * time.Second)

	// F. Speichern & Event senden
	if err := s.Repo.UpdateGame(game); err != nil {
		return nil, err
	}

	s.EventBus.Publish("dice_toggled", game)

	return game, nil
}
