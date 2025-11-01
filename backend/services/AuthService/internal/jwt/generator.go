package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"log/slog"
)

// Generator creates signed JWT tokens for guest users.
// Claims:
//  sub  -> user id
//  name -> username
//  iat  -> issued at (unix)
//  exp  -> expiry (unix) (24h)
//  iss  -> knuffel-auth-service
// Signed with HS256 using secret from environment variable JWT_SECRET.

type Generator struct {
	secret []byte
	issuer string
}

// NewGenerator builds a new Generator. If secret argument empty, it falls back to env var JWT_SECRET.
func NewGenerator(secret string) *Generator {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		logger.Default().Warn("jwt secret not configured during generator initialization")
	}
	return &Generator{secret: []byte(secret), issuer: "knuffel-auth-service"}
}

// CreateToken returns a signed JWT string or error if secret missing or signing fails.
func (g *Generator) CreateToken(userID, username string) (string, error) {
	if len(g.secret) == 0 {
		logger.Default().Error("token generation failed: secret missing", slog.String("user_id", userID))
		return "", errors.New("jwt secret not configured")
	}
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(24 * time.Hour)

	claims := jwtlib.MapClaims{
		"sub":   userID,
		"name":  username,
		"iat":   issuedAt.Unix(),
		"exp":   expiresAt.Unix(),
		"iss":   g.issuer,
		"guest": true,
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		logger.Default().Error("token signing failed", slog.String("user_id", userID), slog.String("error", err.Error()))
		return "", err
	}
	logger.Default().Debug("token created", slog.String("user_id", userID), slog.String("username", username))
	return signed, nil
}
