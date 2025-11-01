package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/models"
)

const maxBodySize = 1 << 20 // 1MB

// CreateTokenHandler returns an http.HandlerFunc bound to a JWT generator.
// Requires user_id and username. Guest tokens only; guest claim always true.
func CreateTokenHandler(gen *jwt.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "create_token"))
		var req models.CreateJWTRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			log.Warn("decode failed", slog.String("error", err.Error()))
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}
		errMap := req.Validate()
		if len(errMap) > 0 {
			log.Info("validation failed", slog.Any("errors", errMap))
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Validation failed", map[string]interface{}{"fields": errMap}, log)
			return
		}
		// Always guest tokens from this service
		token, err := gen.CreateToken(req.UserID, req.Username, true)
		if err != nil {
			log.Error("token generation failed", slog.String("error", err.Error()), slog.String("user_id", req.UserID))
			httpx.WriteError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate JWT token", map[string]interface{}{"detail": err.Error()}, log)
			return
		}
		log.Info("token generated", slog.String("user_id", req.UserID), slog.String("username", req.Username), slog.Bool("guest", true))
		httpx.WriteJSON(w, http.StatusOK, models.CreateTokenResponse{Token: token}, log)
	}
}
