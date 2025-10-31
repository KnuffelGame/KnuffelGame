package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/models"
	"github.com/google/uuid"
	"log/slog"
)

// UUID v4 pattern (loosely validates general UUID format w/o version bits enforcement via regex; actual validation uses uuid.Parse)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// CreateTokenHandler returns an http.HandlerFunc bound to a JWT generator.
func CreateTokenHandler(gen *jwt.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("operation", "create_token"))
		var req models.CreateTokenRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			log.Warn("decode failed", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", map[string]interface{}{"detail": err.Error()})
			return
		}

		uname := strings.TrimSpace(req.Username)
		if uname == "" {
			log.Info("validation failed: empty username")
			writeError(w, http.StatusBadRequest, "invalid_request", "Missing required field: username", nil)
			return
		}
		if len(uname) < 3 || len(uname) > 20 {
			log.Info("validation failed: username length", slog.Int("length", len(uname)))
			writeError(w, http.StatusBadRequest, "invalid_request", "Username must be between 3 and 20 characters", map[string]interface{}{"min": 3, "max": 20})
			return
		}

		userID := strings.TrimSpace(req.UserID)
		if userID == "" {
			userID = uuid.New().String()
			log.Debug("generated user id", slog.String("user_id", userID))
		} else {
			if !uuidPattern.MatchString(userID) {
				log.Info("validation failed: pattern", slog.String("user_id", userID))
				writeError(w, http.StatusBadRequest, "invalid_request", "user_id must be a UUID", map[string]interface{}{"pattern": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"})
				return
			}
			if _, err := uuid.Parse(userID); err != nil {
				log.Info("validation failed: parse uuid", slog.String("error", err.Error()))
				writeError(w, http.StatusBadRequest, "invalid_request", "Invalid UUID format", map[string]interface{}{"detail": err.Error()})
				return
			}
		}

		token, err := gen.CreateToken(userID, uname)
		if err != nil {
			log.Error("token generation failed", slog.String("error", err.Error()), slog.String("user_id", userID))
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate JWT token", map[string]interface{}{"detail": err.Error()})
			return
		}

		log.Info("token generated", slog.String("user_id", userID), slog.String("username", uname))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.CreateTokenResponse{Token: token})
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: code, Message: msg, Details: details})
}
