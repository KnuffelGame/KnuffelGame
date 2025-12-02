package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// ConnectionStatsResponse provides aggregate registry stats split by target type.
type ConnectionStatsResponse struct {
	Timestamp        string `json:"timestamp"`
	TotalTargets     int    `json:"total_targets"`
	TotalConnections int    `json:"total_connections"`
	LobbyTargets     int    `json:"lobby_targets"`
	LobbyConnections int    `json:"lobby_connections"`
	GameTargets      int    `json:"game_targets"`
	GameConnections  int    `json:"game_connections"`
}

// GetConnectionStats handles GET /internal/connections
func GetConnectionStats(reg *models.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := requestLogger(r.Context(), nil).WithGroup("internal").With(slog.String("action", "stats"))

		totalTargets, totalConns, lobbyTargets, lobbyConns, gameTargets, gameConns := reg.Stats()
		resp := ConnectionStatsResponse{
			Timestamp:        time.Now().UTC().Format(time.RFC3339),
			TotalTargets:     totalTargets,
			TotalConnections: totalConns,
			LobbyTargets:     lobbyTargets,
			LobbyConnections: lobbyConns,
			GameTargets:      gameTargets,
			GameConnections:  gameConns,
		}
		httpx.WriteJSON(w, http.StatusOK, resp, log)
	}
}
