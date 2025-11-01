package jwt

import (
	"errors"
	"os"
	"time"

	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    string
	Username  string
	Issuer    string
	IssuedAt  time.Time
	ExpiresAt time.Time
	IsGuest   bool
}

type Validator struct {
	secret []byte
	issuer string
}

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTokenExpired     = errors.New("token expired")
	ErrMalformedToken   = errors.New("invalid format")
)

func NewValidator(secret string) *Validator {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		logger.Default().Warn("jwt secret not configured during validator initialization")
	}
	return &Validator{secret: []byte(secret), issuer: "knuffel-auth-service"}
}

func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMalformedToken
	}
	parsedToken, err := jwtlib.ParseWithClaims(tokenString, jwtlib.MapClaims{}, func(t *jwtlib.Token) (interface{}, error) {
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
	claimsMap, ok := parsedToken.Claims.(jwtlib.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, ErrMalformedToken
	}
	issRaw, okIss := claimsMap["iss"].(string)
	if !okIss || issRaw != v.issuer {
		return nil, errors.New("invalid issuer")
	}
	subRaw, okSub := claimsMap["sub"].(string)
	nameRaw, okName := claimsMap["name"].(string)
	if !okSub || subRaw == "" || !okName || nameRaw == "" {
		return nil, errors.New("missing claims")
	}
	var issuedAt, expiresAt time.Time
	if iatFloat, okIat := claimsMap["iat"].(float64); okIat {
		issuedAt = time.Unix(int64(iatFloat), 0)
	}
	if expFloat, okExp := claimsMap["exp"].(float64); okExp {
		expiresAt = time.Unix(int64(expFloat), 0)
		if time.Now().After(expiresAt) {
			return nil, ErrTokenExpired
		}
	}
	isGuest := true
	if guestRaw, okGuest := claimsMap["guest"].(bool); okGuest {
		isGuest = guestRaw
	}
	logger.Default().Debug("token validated", slog.String("user_id", subRaw), slog.String("username", nameRaw))
	return &Claims{UserID: subRaw, Username: nameRaw, Issuer: issRaw, IssuedAt: issuedAt, ExpiresAt: expiresAt, IsGuest: isGuest}, nil
}
