package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/db"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
)

var (
	ErrGameNotFound  = errors.New("game_not_found")
	ErrNotYourTurn   = errors.New("not_your_turn")
	ErrMaxRolls      = errors.New("max_rolls_reached")
	ErrGameNotActive = errors.New("game_not_active")
	ErrInternal      = errors.New("internal_error")
)

// GameService ist die "Klasse", die die Geschäftslogik kapselt.
type GameService struct {
	Repo *db.Repository
}

// NewGameService ist der Konstruktor
func NewGameService(repo *db.Repository) *GameService {
	return &GameService{
		Repo: repo,
	}
}

// RollDice führt den Würfelvorgang durch
func (s *GameService) RollDice(ctx context.Context, gameID string, userID string) (*models.RollDiceResponse, error) {
	// DEBUGGING MIT LOG
	if s == nil {
		log.Println("CRITICAL: 's' (GameService) ist nil!")
		return nil, errors.New("service is nil")
	}
	if s.Repo == nil {
		log.Println("CRITICAL: 's.Repo' ist nil! Datenbank-Verbindung fehlt im Service!")
		return nil, errors.New("repo is nil")
	}

	log.Println("DEBUG: Service und Repo sind da. Starte DB-Abfrage...")

	// 1. Spiel aus der echten DB laden
	game, err := s.Repo.GetGameByID(ctx, gameID)
	if err != nil {
		// Fehlerbehandlung: Repository gibt ggf. spezifische Fehler zurück
		return nil, err
	}

	// 2. Validierung: Spiel-Status
	if game.Status != "active" {
		return nil, ErrGameNotActive
	}

	// 3. Validierung: Ist der User dran?
	// Da TurnOrder als JSONB in der DB liegt, müssen wir es parsen
	var turnOrder []models.TurnOrderEntry
	if err := json.Unmarshal(game.TurnOrder, &turnOrder); err != nil {
		return nil, ErrInternal // Dateninkonsistenz in der DB
	}

	if len(turnOrder) == 0 {
		return nil, ErrInternal
	}

	// Berechne, wer dran ist. Annahme: CurrentTurn ist 1-basiert (Runde 1, Zug 1...)
	// (CurrentTurn - 1) % AnzahlSpieler ergibt den Index im Array
	playerIndex := (game.CurrentTurn - 1) % len(turnOrder)
	expectedPlayerID := turnOrder[playerIndex].UserID

	if expectedPlayerID != userID {
		return nil, ErrNotYourTurn
	}

	// 4. Den aktuellen Turn (Spielzug) aus der DB laden
	currentTurn, err := s.Repo.GetCurrentTurn(ctx, gameID, userID)
	if err != nil {
		return nil, err
	}

	if currentTurn == nil {
		// Initialisiere ein neues Turn-Objekt
		currentTurn = &models.TurnDB{
			GameID:     gameID,
			UserID:     userID,
			RollCount:  0,               // Noch 0, wird gleich erhöht
			DiceValues: make([]int, 5),  // Leeres Array vorbereiten
			KeptDice:   make([]bool, 5), // Alles false
			StartedAt:  time.Now(),
		}

		// Optional: Wenn du UUIDs für Turns brauchst, generiere sie hier:
		// currentTurn.ID = uuid.NewString()

		// WICHTIG: Du musst sicherstellen, dass dein Repository diesen neuen Turn auch speichert.
		// Entweder rufst du hier explizit CreateTurn auf:
		if err := s.Repo.CreateTurn(ctx, currentTurn); err != nil {
			return nil, err
		}
	}

	// 5. Validierung: Maximale Würfe
	if currentTurn.RollCount >= 3 {
		return nil, ErrMaxRolls
	}

	// --- LOGIK: WÜRFELN ---
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Falls DiceValues leer ist (erster Wurf)
	if len(currentTurn.DiceValues) != 5 {
		currentTurn.DiceValues = make([]int, 5)
		currentTurn.KeptDice = make([]bool, 5)
	}

	for i := 0; i < 5; i++ {
		// Nur würfeln, wenn der Würfel nicht gesperrt ist
		if !currentTurn.KeptDice[i] {
			currentTurn.DiceValues[i] = r.Intn(6) + 1
		}
	}

	// Aktualisierungen am Objekt
	currentTurn.RollCount++
	currentTurn.StartedAt = time.Now() // Reset Timeout Timer

	// 6. Speichern in der DB
	if err := s.Repo.UpdateTurn(ctx, currentTurn); err != nil {
		return nil, err
	}

	// 7. Response bauen (Mapping von DB-Modell auf API-Modell)
	apiDice := make([]models.Dice, 5)
	for i := 0; i < 5; i++ {
		apiDice[i] = models.Dice{
			Value:  currentTurn.DiceValues[i],
			Locked: currentTurn.KeptDice[i],
		}
	}

	return &models.RollDiceResponse{
		GameID:          game.ID,
		RollCount:       currentTurn.RollCount,
		Dice:            apiDice,
		CanRollAgain:    currentTurn.RollCount < 3,
		MustSelectField: currentTurn.RollCount >= 3,
	}, nil
}
