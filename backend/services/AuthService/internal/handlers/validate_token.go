package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/models"
)

const maxValidateBodySize = 1 << 20 // 1MB

// ValidateTokenHandler validates a JWT and returns success/failure response.
func ValidateTokenHandler(validator *jwt.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxValidateBodySize)
		log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "validate_token"))
		var req models.ValidateJWTRequest
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
			httpx.WriteJSON(w, http.StatusBadRequest, models.ValidateTokenResponse{Valid: false, Error: "invalid format"}, log)
			return
		}
		claims, err := validator.ValidateToken(req.Token)
		if err != nil {
			errorCode := mapValidationError(err)
			status := http.StatusUnauthorized
			if err == jwt.ErrMalformedToken {
				status = http.StatusBadRequest
			}
			log.Info("token invalid", slog.String("error", errorCode))
			httpx.WriteJSON(w, status, models.ValidateTokenResponse{Valid: false, Error: errorCode}, log)
			return
		}
		log.Info("token valid", slog.String("user_id", claims.Subject), slog.String("username", claims.Username))
		httpx.WriteJSON(w, http.StatusOK, models.ValidateTokenResponse{Valid: true, UserID: claims.Subject, Username: claims.Username, IsGuest: claims.Guest}, log)
	}
}

func mapValidationError(err error) string {
	switch err {
	case jwt.ErrInvalidSignature:
		return "invalid signature"
	case jwt.ErrTokenExpired:
		return "token expired"
	case jwt.ErrMalformedToken:
		return "invalid format"
	case jwt.ErrInvalidIssuer:
		return "invalid issuer"
	case jwt.ErrMissingClaims:
		return "missing claims"
	default:
		return "invalid format"
	}
}
