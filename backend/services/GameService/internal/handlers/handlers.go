package handlers

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/db"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	repo       *db.Repository
	logger     *slog.Logger
	diceEngine *services.GameService
}

func NewHandler(repo *db.Repository, logger *slog.Logger, diceEngine *services.GameService) *Handler {
	return &Handler{
		repo:       repo,
		logger:     logger,
		diceEngine: diceEngine,
	}
}

func (h *Handler) CreateGame(c *gin.Context) {
	var req models.CreateGameRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	gameID := uuid.New().String()
	startTime := time.Now()
	firstPlayer := req.TurnOrder[0]
	turnOrder := req.TurnOrder

	categories := []string{
		"ones", "twos", "threes", "fours", "fives", "sixes", "bonus",
		"three_of_a_kind", "four_of_a_kind", "full_house",
		"small_straight", "large_straight", "kniffel", "chance",
	}

	// Context für die DB-Operationen holen
	ctx := c.Request.Context()

	err := h.repo.WithTransaction(ctx, func(txRepo *db.Repository) error {

		// A. Spiel anlegen
		// Wir erstellen das GameDB Objekt.
		// Achtung: Ich caste turnOrderJSON zu string, da viele SQL-Treiber das für JSONB bevorzugen.
		game := models.GameDB{
			ID:          gameID,
			LobbyID:     req.LobbyID,
			Status:      "active",
			CurrentTurn: 1,
			TurnOrder:   turnOrder, // Hier wird das JSON gespeichert
			Round:       1,
			StartedAt:   startTime,
		}

		if err := txRepo.CreateGame(ctx, &game); err != nil {
			return fmt.Errorf("failed to create game: %w", err)
		}

		// B. Ersten Turn anlegen
		for _, player := range req.TurnOrder {
			firstTurn := models.TurnDB{
				ID:         uuid.New().String(),
				GameID:     gameID,
				UserID:     player.PlayerID,
				RollCount:  0,
				DiceValues: []int{0, 0, 0, 0, 0}, // Wichtig: Leeres Array, nicht nil
				KeptDice:   []bool{false, false, false, false, false},
				Timeout:    false,
				StartedAt:  startTime,
				Round:      1,
			}

			if err := txRepo.CreateTurn(ctx, &firstTurn); err != nil {
				return fmt.Errorf("failed to create initial turn: %w", err)
			}
		}

		// C. Scorecards für ALLE Spieler anlegen
		// Wir iterieren über jeden Spieler und für jeden Spieler über jede Kategorie.
		for _, player := range req.TurnOrder {
			for _, category := range categories {
				scorecard := models.ScorecardDB{
					ID:          uuid.New().String(),
					GameID:      gameID,
					UserID:      player.PlayerID,
					FieldName:   category,
					Value:       0,   // Startwert
					RoundFilled: nil, // WICHTIG: nil bedeutet "noch offen"
				}

				if err := txRepo.CreateScorecard(ctx, &scorecard); err != nil {
					return err
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
		CurrentPlayerID: firstPlayer.PlayerID,
		TurnOrder:       turnOrder, // Wir geben die Struktur zurück, Gin macht daraus JSON
	}

	h.logger.Info("Game created successfully", "game_id", gameID, "lobby_id", req.LobbyID)
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) PostRollDice(c *gin.Context) {
	gameID := c.Param("game_id")
	userID := c.GetHeader("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing User-Id header"})
		return
	}

	// Service Aufruf
	response, err := h.diceEngine.RollDice(c, gameID, userID)

	if err != nil {
		switch err {
		case models.ErrGameNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})

		case models.ErrNotYourTurn:
			// OpenAPI: 403 Forbidden - Not current player
			// In einer echten App würdest du hier den "current_player" Namen aus der DB holen
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "It's not your turn",
				Details: models.ErrorDetails{
					CurrentPlayer: "other_player_id", // Hier den echten ID einsetzen
				},
			})

		case models.ErrMaxRolls:
			// OpenAPI: 403 Forbidden - Max rolls reached
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Maximum rolls (3) reached - must select a field",
				Details: models.ErrorDetails{
					RollCount: 3,
				},
			})

		case models.ErrGameNotActive:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Game is not running"})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	// 200 OK
	c.JSON(http.StatusOK, response)
}

func (h *Handler) PostSelectDice(c *gin.Context) {
	gameID := c.Param("game_id")
	userID := c.GetHeader("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing User-Id header"})
		return
	}

	var req models.ToggleDiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	response, err := h.diceEngine.ToggleDice(c, gameID, userID, req.DiceIndices)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrGameNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		case errors.Is(err, models.ErrNotYourTurn):
			c.JSON(http.StatusForbidden, gin.H{"error": "It's not your turn"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) PostSelectScoreField(c *gin.Context) {
	gameID := c.Param("game_id")
	userID := c.GetHeader("user_id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing User-Id header"})
		return
	}

	var req models.SelectScoreFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request payload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	log.Println("starting select score field")
	response, err := h.diceEngine.SelectScoreField(c, gameID, userID, req.FieldName)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrGameNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		case errors.Is(err, models.ErrNotYourTurn):
			c.JSON(http.StatusForbidden, gin.H{"error": "It's not your turn"})
		case errors.Is(err, models.ErrFieldAlreadySelected):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field already selected"})
		default:
			c.JSON(http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetGameState(c *gin.Context) {
	gameID := c.Param("game_id")

	response, err := h.diceEngine.BuildGameState(c, gameID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrGameNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		default:
			c.JSON(http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, response)
}
