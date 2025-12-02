package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
	"github.com/go-chi/chi/v5"
)

// AuthorizeLobbyMembership ensures the authenticated user is a member of the lobby.
// It calls APIGateway GET /lobbies/{lobby_id} with the client's JWT cookie to verify membership.
// Timeouts: 3s; retries: none.
func AuthorizeLobbyMembership(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
	client := &http.Client{Timeout: 3 * time.Second}
	cookieName := "jwt"
	if cfg != nil && cfg.JWTCookieName != "" {
		cookieName = cfg.JWTCookieName
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(
				slog.String("action", "authorize_lobby"),
			)
			if baseLog != nil {
				log = baseLog.WithGroup("middleware").With(slog.String("action", "authorize_lobby"))
			}

			if cfg == nil || cfg.APIGatewayBaseURL == "" {
				log.Error("APIGatewayBaseURL missing; cannot authorize lobby membership")
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}

			lobbyID := chi.URLParam(r, "lobby_id")
			if lobbyID == "" {
				handlers.BadRequest(w, "Invalid lobby_id format", map[string]interface{}{"lobby_id": lobbyID}, log)
				return
			}

			userID := r.Header.Get("X-User-ID")
			username := r.Header.Get("X-Username")
			if userID == "" || username == "" {
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}
			// Forward the original JWT cookie for gateway auth
			var cookieVal string
			if c, err := r.Cookie(cookieName); err == nil && c != nil {
				cookieVal = c.Value
			}

			req, err := http.NewRequest(http.MethodGet, cfg.APIGatewayBaseURL+"/lobbies/"+lobbyID, nil)
			if err != nil {
				log.Error("request build failed", slog.String("error", err.Error()))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
			req.Header.Set("Accept", "application/json")
			if cookieVal != "" {
				req.Header.Set("Cookie", cookieName+"="+cookieVal)
			}
			// propagate request-id if present
			if rid := r.Header.Get("X-Request-ID"); rid != "" {
				req.Header.Set("X-Request-ID", rid)
			}
			// propagate internal token if configured/required
			if cfg.InternalToken != "" {
				req.Header.Set("X-Internal-Token", cfg.InternalToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Error("gateway call failed", slog.String("error", err.Error()))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				// Optionally parse players to verify membership defensively
				var body struct {
					Players []struct {
						UserID string `json:"user_id"`
					} `json:"players"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && len(body.Players) > 0 {
					found := false
					for _, p := range body.Players {
						if p.UserID == userID {
							found = true
							break
						}
					}
					if !found {
						log.Info("membership check failed (not in players array)", slog.String("user_id", userID), slog.String("lobby_id", lobbyID))
						handlers.Forbidden(w, "You are not a member of this lobby", log)
						return
					}
				}
				log.Info("lobby membership authorized", slog.String("lobby_id", lobbyID), slog.String("user_id", userID))
				next.ServeHTTP(w, r)
				return
			case http.StatusForbidden:
				log.Info("lobby membership forbidden", slog.String("lobby_id", lobbyID), slog.String("user_id", userID))
				handlers.Forbidden(w, "You are not a member of this lobby", log)
				return
			case http.StatusNotFound:
				log.Info("lobby not found", slog.String("lobby_id", lobbyID))
				handlers.NotFound(w, "Lobby not found", log)
				return
			case http.StatusUnauthorized:
				log.Info("unauthorized at gateway during membership check")
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			default:
				log.Error("unexpected gateway status", slog.Int("status", resp.StatusCode))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
		})
	}
}

// AuthorizeGameMembership ensures the authenticated user is a player in the game.
// It calls APIGateway GET /games/{game_id} with the client's JWT cookie to verify membership.
// Timeouts: 3s; retries: none.
func AuthorizeGameMembership(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
	client := &http.Client{Timeout: 3 * time.Second}
	cookieName := "jwt"
	if cfg != nil && cfg.JWTCookieName != "" {
		cookieName = cfg.JWTCookieName
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Logger(r.Context()).WithGroup("middleware").With(
				slog.String("action", "authorize_game"),
			)
			if baseLog != nil {
				log = baseLog.WithGroup("middleware").With(slog.String("action", "authorize_game"))
			}

			if cfg == nil || cfg.APIGatewayBaseURL == "" {
				log.Error("APIGatewayBaseURL missing; cannot authorize game membership")
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}

			gameID := chi.URLParam(r, "game_id")
			if gameID == "" {
				handlers.BadRequest(w, "Invalid game_id format", map[string]interface{}{"game_id": gameID}, log)
				return
			}

			userID := r.Header.Get("X-User-ID")
			username := r.Header.Get("X-Username")
			if userID == "" || username == "" {
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			}

			// Forward the original JWT cookie for gateway auth
			var cookieVal string
			if c, err := r.Cookie(cookieName); err == nil && c != nil {
				cookieVal = c.Value
			}

			req, err := http.NewRequest(http.MethodGet, cfg.APIGatewayBaseURL+"/games/"+gameID, nil)
			if err != nil {
				log.Error("request build failed", slog.String("error", err.Error()))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
			req.Header.Set("Accept", "application/json")
			if cookieVal != "" {
				req.Header.Set("Cookie", cookieName+"="+cookieVal)
			}
			// propagate request-id if present
			if rid := r.Header.Get("X-Request-ID"); rid != "" {
				req.Header.Set("X-Request-ID", rid)
			}
			// propagate internal token if configured/required
			if cfg.InternalToken != "" {
				req.Header.Set("X-Internal-Token", cfg.InternalToken)
			}

			resp, err := client.Do(req)
			if err != nil {
				log.Error("gateway call failed", slog.String("error", err.Error()))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
			defer resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				log.Info("game membership authorized", slog.String("game_id", gameID), slog.String("user_id", userID))
				next.ServeHTTP(w, r)
				return
			case http.StatusForbidden:
				log.Info("game membership forbidden", slog.String("game_id", gameID), slog.String("user_id", userID))
				handlers.Forbidden(w, "You are not a player in this game", log)
				return
			case http.StatusNotFound:
				log.Info("game not found", slog.String("game_id", gameID))
				handlers.NotFound(w, "Game not found", log)
				return
			case http.StatusUnauthorized:
				log.Info("unauthorized at gateway during membership check")
				handlers.Unauthorized(w, "Invalid or expired authentication token", log)
				return
			default:
				log.Error("unexpected gateway status", slog.Int("status", resp.StatusCode))
				handlers.InternalServerError(w, "Internal server error", nil, log)
				return
			}
		})
	}
}
