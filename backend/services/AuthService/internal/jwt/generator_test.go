package jwt

import (
	"os"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestCreateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret")
	gen := NewGenerator("")

	tokenStr, err := gen.CreateToken("usr_test123", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatalf("expected token string, got empty")
	}

	token, err := jwtlib.Parse(tokenStr, func(token *jwtlib.Token) (interface{}, error) {
		if token.Method.Alg() != jwtlib.SigningMethodHS256.Alg() {
			t.Fatalf("unexpected signing method: %s", token.Method.Alg())
		}
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok || !token.Valid {
		t.Fatalf("invalid token claims")
	}

	if claims["sub"] != "usr_test123" {
		t.Errorf("expected sub usr_test123, got %v", claims["sub"])
	}
	if claims["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", claims["name"])
	}
	if claims["iss"] != "knuffel-auth-service" {
		t.Errorf("expected iss knuffel-auth-service, got %v", claims["iss"])
	}
	// exp should be ~24h from iat
	_iat, ok1 := claims["iat"].(float64)
	_exp, ok2 := claims["exp"].(float64)
	if !ok1 || !ok2 {
		t.Fatalf("iat/exp not numeric")
	}
	if int64(_exp)-int64(_iat) != int64(24*time.Hour/time.Second) {
		t.Errorf("expected exp-iat == 86400, got %d", int64(_exp)-int64(_iat))
	}
	if claims["guest"] != true {
		t.Errorf("expected guest true, got %v", claims["guest"])
	}
}
