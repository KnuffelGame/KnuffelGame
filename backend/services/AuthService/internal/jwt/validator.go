package jwt

import (
	"errors"
	"os"
	"time"

	"log/slog"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Claims represents validated token data returned by Validator
// Only fields needed by downstream services are exposed.
// IsGuest is currently always true for guest tokens (OIDC stretch goal will set false).
// IssuedAt / ExpiresAt retained for potential auditing.

type Claims struct {
	UserID    string
	Username  string
	Issuer    string
	IssuedAt  time.Time
	ExpiresAt time.Time
	IsGuest   bool
}

// Validator validates and parses JWT tokens.
// It expects HS256 signed tokens with issuer "knuffel-auth-service".
// Secret is loaded from argument or environment variable JWT_SECRET.

type Validator struct {
	secret []byte
	issuer string
}

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTokenExpired     = errors.New("token expired")
	ErrMalformedToken   = errors.New("invalid format")
)

// NewValidator constructs a Validator instance.
func NewValidator(secret string) *Validator {
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		logger.Default().Warn("jwt secret not configured during validator initialization")
	}
	return &Validator{secret: []byte(secret), issuer: "knuffel-auth-service"}
}

// ValidateToken parses and validates a JWT string, returning claims or categorized error.
// Error categories mapped to OpenAPI spec: token expired, invalid signature, invalid format, invalid issuer, missing claims.
func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMalformedToken
	}
	parsedToken, err := jwtlib.ParseWithClaims(tokenString, jwtlib.MapClaims{}, func(t *jwtlib.Token) (interface{}, error) {
		// enforce signing method HS256
		if t.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			return nil, ErrInvalidSignature
		}
		return v.secret, nil
	})
	if err != nil {
		// Map jwt library errors to domain errors
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwtlib.ErrTokenSignatureInvalid) || errors.Is(err, jwtlib.ErrTokenUnverifiable) {
			return nil, ErrInvalidSignature
		}
		// Any other parse error treated as malformed
		return nil, ErrMalformedToken
	}

	claimsMap, ok := parsedToken.Claims.(jwtlib.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, ErrMalformedToken
	}

	// issuer check
	issRaw, okIss := claimsMap["iss"].(string)
	if !okIss || issRaw != v.issuer {
		return nil, errors.New("invalid issuer")
	}

	// required claims
	subRaw, okSub := claimsMap["sub"].(string)
	nameRaw, okName := claimsMap["name"].(string)
	if !okSub || subRaw == "" || !okName || nameRaw == "" {
		return nil, errors.New("missing claims")
	}

	// issued at / expiry
	var issuedAt, expiresAt time.Time
	if iatFloat, okIat := claimsMap["iat"].(float64); okIat {
		issuedAt = time.Unix(int64(iatFloat), 0)
	}
	if expFloat, okExp := claimsMap["exp"].(float64); okExp {
		expiresAt = time.Unix(int64(expFloat), 0)
		if time.Now().After(expiresAt) {
			// In rare race, library didn't flag expired yet (edge within clock skew). Treat as expired.
			return nil, ErrTokenExpired
		}
	}
	isGuest := true
	if guestRaw, okGuest := claimsMap["guest"].(bool); okGuest {
		isGuest = guestRaw
	}

	logger.Default().Debug("token validated", slog.String("user_id", subRaw), slog.String("username", nameRaw))
	return &Claims{
		UserID:    subRaw,
		Username:  nameRaw,
		Issuer:    issRaw,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		IsGuest:   isGuest,
	}, nil
}
