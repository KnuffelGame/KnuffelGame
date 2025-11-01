package handlers

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/models"
)

// ValidateTokenHandler validates a JWT and returns success/failure response per OpenAPI spec.
func ValidateTokenHandler(validator *jwt.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("operation", "validate_token"))
		var req models.ValidateJWTRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			log.Warn("decode failed", slog.String("error", err.Error()))
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", map[string]interface{}{"detail": err.Error()})
			return
		}
		errMap := req.Validate()
		if len(errMap) > 0 {
			log.Info("validation failed", slog.Any("errors", errMap))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			valid := false
			_ = json.NewEncoder(w).Encode(models.JWTResponse{Valid: &valid, Error: "invalid format"})
			return
		}

		claims, err := validator.ValidateToken(req.Token)
		if err != nil {
			errorCode := mapValidationError(err)
			log.Info("token invalid", slog.String("error", errorCode))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			valid := false
			_ = json.NewEncoder(w).Encode(models.JWTResponse{Valid: &valid, Error: errorCode})
			return
		}

		log.Info("token valid", slog.String("user_id", claims.UserID), slog.String("username", claims.Username))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		valid := true
		isGuest := claims.IsGuest
		_ = json.NewEncoder(w).Encode(models.JWTResponse{Valid: &valid, UserID: claims.UserID, Username: claims.Username, IsGuest: &isGuest})
	}
}

func mapValidationError(err error) string {
	if err == jwt.ErrInvalidSignature {
		return "invalid signature"
	}
	if err == jwt.ErrTokenExpired {
		return "token expired"
	}
	if err == jwt.ErrMalformedToken {
		return "invalid format"
	}
	if err.Error() == "invalid issuer" {
		return "invalid issuer"
	}
	if err.Error() == "missing claims" {
		return "missing claims"
	}
	return "invalid format"
}
