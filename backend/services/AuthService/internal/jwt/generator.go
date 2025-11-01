package jwt

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	Issuer       = "knuffel-auth-service"
	minSecretLen = 32
)

var (
	ErrSecretMissing = errors.New("jwt secret not configured")
	ErrSecretWeak    = errors.New("jwt secret too short")
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
	log    *slog.Logger
}

// NewGenerator builds a new Generator. If secret argument empty, it falls back to env var JWT_SECRET.
// Warns when secret missing or weak.
func NewGenerator(secret string) *Generator {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	l := logger.Default().WithGroup("jwt").With(slog.String("component", "generator"))
	if secret == "" {
		l.Warn("jwt secret not configured during generator initialization")
	} else if len(secret) < minSecretLen {
		l.Warn("jwt secret length below recommended minimum", slog.Int("length", len(secret)))
	}
	return &Generator{secret: []byte(secret), issuer: Issuer, log: l}
}

// Claims defines the token claims we embed.
type Claims struct {
	Username string `json:"name"`
	Guest    bool   `json:"guest"`
	jwtlib.RegisteredClaims
}

// CreateToken returns a signed JWT string or error if secret missing or signing fails.
func (g *Generator) CreateToken(userID, username string, guest bool) (string, error) {
	if len(g.secret) == 0 {
		g.log.Error("token generation failed: secret missing", slog.String("user_id", userID))
		return "", ErrSecretMissing
	}
	if len(g.secret) < minSecretLen {
		g.log.Error("token generation failed: secret weak", slog.String("user_id", userID))
		return "", ErrSecretWeak
	}
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(24 * time.Hour)
	claims := Claims{
		Username: username,
		Guest:    guest,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			Issuer:    g.issuer,
			IssuedAt:  jwtlib.NewNumericDate(issuedAt),
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		g.log.Error("token signing failed", slog.String("user_id", userID), slog.String("error", err.Error()))
		return "", err
	}
	g.log.Debug("token created", slog.String("user_id", userID), slog.String("username", username))
	return signed, nil
}
