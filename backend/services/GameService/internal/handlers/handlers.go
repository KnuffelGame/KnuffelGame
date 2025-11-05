package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/services/GameService/internal/database"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   *database.Repository
	logger *slog.Logger
}

func NewHandler(repo *database.Repository, logger *slog.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

func (h *Handler) GetGameState(c *gin.Context) {
	gameID := c.Param("game_id")

	if gameID == "" {
		h.logger.Warn("Missing game_id parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing game ID."})
		return
	}

	gameState, err := h.repo.GetGameStateByID(c.Request.Context(), gameID)
	if err != nil {
		// 4. Fehler behandeln
		if errors.Is(err, database.ErrGameNotFound) {
			// Spezifischer Fehler: 404 Not Found
			c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
			return
		}

		// Allgemeiner Fehler: 500 Internal Server Error
		h.logger.Error("Failed to get game state", "error", err, "game_id", gameID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 5. Erfolg: 200 OK und GameState als JSON senden
	c.JSON(http.StatusOK, gameState)
}
