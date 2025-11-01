package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/models"
	"github.com/google/uuid"
	"log/slog"
)

// CreateTokenHandler returns an http.HandlerFunc bound to a JWT generator.
func CreateTokenHandler(gen *jwt.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("operation", "create_token"))
		var req models.CreateJWTRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			log.Warn("decode failed", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", map[string]interface{}{"detail": err.Error()})
			return
		}
		errMap := req.Validate()
		if len(errMap) > 0 {
			log.Info("validation failed", slog.Any("errors", errMap))
			writeError(w, http.StatusBadRequest, "invalid_request", "Validation failed", map[string]interface{}{"fields": errMap})
			return
		}

		userID := req.UserID
		if userID == "" { // generate uuid if not provided
			userID = uuid.New().String()
			log.Debug("generated user id", slog.String("user_id", userID))
		}
		uname := req.Username

		token, err := gen.CreateToken(userID, uname)
		if err != nil {
			log.Error("token generation failed", slog.String("error", err.Error()), slog.String("user_id", userID))
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate JWT token", map[string]interface{}{"detail": err.Error()})
			return
		}

		log.Info("token generated", slog.String("user_id", userID), slog.String("username", uname))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.JWTResponse{Token: token})
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: code, Message: msg, Details: details})
}
