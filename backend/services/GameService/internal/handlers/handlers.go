package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/db"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	repo   *db.Repository
	logger *slog.Logger
}

func NewHandler(repo *db.Repository, logger *slog.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

//func (h *Handler) GetGameState(c *gin.Context) {
//	gameID := c.Param("game_id")
//
//	if gameID == "" {
//		h.logger.Warn("Missing game_id parameter")
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing game ID."})
//		return
//	}
//
//	gameState, err := h.repo.GetGameStateByID(c.Request.Context(), gameID)
//	if err != nil {
//		// 4. Fehler behandeln
//		if errors.Is(err, db.ErrGameNotFound) {
//			// Spezifischer Fehler: 404 Not Found
//			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
//			return
//		}
//
//		// Allgemeiner Fehler: 500 Internal Server Error
//		h.logger.Error("Failed to get game state", "error", err, "game_id", gameID)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
//		return
//	}
//
//	// 5. Erfolg: 200 OK und GameState als JSON senden
//	c.JSON(http.StatusOK, gameState)
//}

func (h *Handler) CreateGame(c *gin.Context) {
	// TODO
	// create GameState, TurnOrder
	// insert Scoreboards for all players
	var req models.CreateGameRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	gameID := uuid.New().String()
	startTime := time.Now()
	firstPlayer := req.TurnOrder[0]

	turnOrderJSON, err := json.Marshal(req.TurnOrder)
	if err != nil {
		h.logger.Error("Failed to marshal turn order", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	categories := []string{
		"ones", "twos", "threes", "fours", "fives", "sixes",
		"three_of_a_kind", "four_of_a_kind", "full_house",
		"small_straight", "large_straight", "kniffel", "chance",
	}

	// Context für die DB-Operationen holen
	ctx := c.Request.Context()

	err = h.repo.WithTransaction(ctx, func(txRepo *db.Repository) error {

		// A. Spiel anlegen
		// Wir erstellen das GameDB Objekt.
		// Achtung: Ich caste turnOrderJSON zu string, da viele SQL-Treiber das für JSONB bevorzugen.
		game := models.GameDB{
			ID:          gameID,
			LobbyID:     req.LobbyID,
			Status:      "active",
			CurrentTurn: 0,
			TurnOrder:   turnOrderJSON, // Hier wird das JSON gespeichert
			Round:       1,
			StartedAt:   startTime,
		}

		if err := txRepo.CreateGame(ctx, &game); err != nil {
			return fmt.Errorf("failed to create game: %w", err)
		}

		// B. Ersten Turn anlegen
		// Initialer Zustand: 0 Würfe, alle Würfel auf 0, nichts gehalten.
		firstTurn := models.TurnDB{
			ID:         uuid.New().String(),
			GameID:     gameID,
			UserID:     firstPlayer.UserID,
			RollCount:  0,
			DiceValues: []int{0, 0, 0, 0, 0}, // Wichtig: Leeres Array, nicht nil
			KeptDice:   []bool{false, false, false, false, false},
			Timeout:    false,
			StartedAt:  startTime,
		}

		if err := txRepo.CreateTurn(ctx, &firstTurn); err != nil {
			return fmt.Errorf("failed to create initial turn: %w", err)
		}

		// C. Scorecards für ALLE Spieler anlegen
		// Wir iterieren über jeden Spieler und für jeden Spieler über jede Kategorie.
		for _, player := range req.TurnOrder {
			for _, category := range categories {
				scorecard := models.ScorecardDB{
					ID:          uuid.New().String(),
					GameID:      gameID,
					UserID:      player.UserID,
					FieldName:   category,
					Value:       0,   // Startwert
					RoundFilled: nil, // WICHTIG: nil bedeutet "noch offen"
				}

				if err := txRepo.CreateScorecard(ctx, &scorecard); err != nil {
					return fmt.Errorf("failed to create scorecard for user %s field %s: %w", player.UserID, category, err)
				}
			}
		}

		// Wenn wir hier ankommen, wird automatisch committed.
		return nil
	})

	// 4. FEHLERBEHANDLUNG DER TRANSAKTION
	if err != nil {
		h.logger.Error("Transaction failed during CreateGame", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create game"})
		return
	}

	// 5. ERFOLGREICHE ANTWORT
	// Wir senden die GameID und den ersten Spieler zurück, damit der Client weiß, wie es losgeht.
	response := models.CreateGameResponse{
		GameID:          gameID,
		LobbyID:         req.LobbyID,
		CurrentPlayerID: firstPlayer.UserID,
		TurnOrder:       req.TurnOrder, // Wir geben die Struktur zurück, Gin macht daraus JSON
	}

	h.logger.Info("Game created successfully", "game_id", gameID, "lobby_id", req.LobbyID)
	c.JSON(http.StatusCreated, response)
}
