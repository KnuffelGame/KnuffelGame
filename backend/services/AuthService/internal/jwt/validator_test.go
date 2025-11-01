package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func makeToken(secret string, claims jwtlib.MapClaims) string {
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	str, _ := tok.SignedString([]byte(secret))
	return str
}

func TestValidateToken_Success(t *testing.T) {
	secret := "test-secret"
	gen := NewGenerator(secret)
	validator := NewValidator(secret)
	jwtStr, err := gen.CreateToken("usr_val123", "Bob")
	if err != nil {
		t.Fatalf("generator error: %v", err)
	}
	claims, err := validator.ValidateToken(jwtStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "usr_val123" || claims.Username != "Bob" || !claims.IsGuest {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	validator := NewValidator(secret)
	claims := jwtlib.MapClaims{
		"sub":  "usr_expired",
		"name": "Eve",
		"iat":  time.Now().Add(-25 * time.Hour).Unix(),
		"exp":  time.Now().Add(-24 * time.Hour).Unix(),
		"iss":  "knuffel-auth-service",
	}
	tok := makeToken(secret, claims)
	_, err := validator.ValidateToken(tok)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	validator := NewValidator(secret)
	claims := jwtlib.MapClaims{
		"sub":  "usr_sig",
		"name": "Mallory",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iss":  "knuffel-auth-service",
	}
	// sign with different secret
	tok := makeToken("other-secret", claims)
	_, err := validator.ValidateToken(tok)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	secret := "test-secret"
	validator := NewValidator(secret)
	_, err := validator.ValidateToken("not-a-jwt")
	if err != ErrMalformedToken {
		t.Fatalf("expected ErrMalformedToken, got %v", err)
	}
}

func TestValidateToken_InvalidIssuer(t *testing.T) {
	secret := "test-secret"
	validator := NewValidator(secret)
	claims := jwtlib.MapClaims{
		"sub":  "usr_issuer",
		"name": "Issuer",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iss":  "other-issuer",
	}
	tok := makeToken(secret, claims)
	_, err := validator.ValidateToken(tok)
	if err == nil || err.Error() != "invalid issuer" {
		t.Fatalf("expected invalid issuer error, got %v", err)
	}
}

func TestValidateToken_MissingClaims(t *testing.T) {
	secret := "test-secret"
	validator := NewValidator(secret)
	claims := jwtlib.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iss": "knuffel-auth-service",
	}
	tok := makeToken(secret, claims)
	_, err := validator.ValidateToken(tok)
	if err == nil || err.Error() != "missing claims" {
		t.Fatalf("expected missing claims error, got %v", err)
	}
}
