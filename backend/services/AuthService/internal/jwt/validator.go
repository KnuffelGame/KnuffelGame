package jwt

import (
	"errors"
	"log/slog"
	"os"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTokenExpired     = errors.New("token expired")
	ErrMalformedToken   = errors.New("invalid format")
	ErrInvalidIssuer    = errors.New("invalid issuer")
	ErrMissingClaims    = errors.New("missing claims")
)

type Validator struct {
	secret []byte
	issuer string
	log    *slog.Logger
}

// NewValidator builds validator; warns on missing/weak secret.
func NewValidator(secret string) *Validator {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	l := logger.Default().WithGroup("jwt").With(slog.String("component", "validator"))
	if secret == "" {
		l.Warn("jwt secret not configured during validator initialization")
	} else if len(secret) < minSecretLen {
		l.Warn("jwt secret length below recommended minimum", slog.Int("length", len(secret)))
	}
	return &Validator{secret: []byte(secret), issuer: Issuer, log: l}
}

func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMalformedToken
	}
	parsedToken, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(t *jwtlib.Token) (interface{}, error) {
		if t.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			return nil, ErrInvalidSignature
		}
		return v.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwtlib.ErrTokenSignatureInvalid) || errors.Is(err, jwtlib.ErrTokenUnverifiable) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrMalformedToken
	}
	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, ErrMalformedToken
	}
	if claims.Issuer != v.issuer {
		return nil, ErrInvalidIssuer
	}
	if claims.Subject == "" || claims.Username == "" {
		return nil, ErrMissingClaims
	}
	v.log.Debug("token validated", slog.String("user_id", claims.Subject), slog.String("username", claims.Username))
	return claims, nil
}
