package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
	"github.com/go-chi/chi/v5"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

func makeRouter(val *jwt.Validator) http.Handler {
	r := chi.NewRouter()
	r.Post("/internal/validate", handlers.ValidateTokenHandler(val))
	return r
}

func TestValidateTokenHandler_Success(t *testing.T) {
	gen := jwt.NewGenerator("secret")
	val := jwt.NewValidator("secret")
	r := makeRouter(val)
	tok, err := gen.CreateToken("usr_valid", "Alice")
	if err != nil {
		t.Fatalf("token create error: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"token": tok})
	req := httptest.NewRequest(http.MethodPost, "/internal/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["valid"] != true || resp["user_id"] != "usr_valid" || resp["username"] != "Alice" || resp["is_guest"] != true {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestValidateTokenHandler_InvalidSignature(t *testing.T) {
	val := jwt.NewValidator("secret")
	r := makeRouter(val)
	claims := jwtlib.MapClaims{"sub": "usr_sig", "name": "Mallory", "iat": time.Now().Unix(), "exp": time.Now().Add(24 * time.Hour).Unix(), "iss": "knuffel-auth-service"}
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	badToken, _ := tok.SignedString([]byte("other"))
	body, _ := json.Marshal(map[string]string{"token": badToken})
	req := httptest.NewRequest(http.MethodPost, "/internal/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["valid"] != false || resp["error"] != "invalid signature" {
		t.Fatalf("expected invalid signature, got %+v", resp)
	}
}

func TestValidateTokenHandler_Malformed(t *testing.T) {
	val := jwt.NewValidator("secret")
	r := makeRouter(val)
	body, _ := json.Marshal(map[string]string{"token": "abc"})
	req := httptest.NewRequest(http.MethodPost, "/internal/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["valid"] != false || resp["error"] != "invalid format" {
		t.Fatalf("expected invalid format, got %+v", resp)
	}
}

func TestValidateTokenHandler_Expired(t *testing.T) {
	val := jwt.NewValidator("secret")
	r := makeRouter(val)
	claims := jwtlib.MapClaims{"sub": "usr_exp", "name": "Bob", "iat": time.Now().Add(-25 * time.Hour).Unix(), "exp": time.Now().Add(-24 * time.Hour).Unix(), "iss": "knuffel-auth-service"}
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	expiredToken, _ := tok.SignedString([]byte("secret"))
	body, _ := json.Marshal(map[string]string{"token": expiredToken})
	req := httptest.NewRequest(http.MethodPost, "/internal/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["valid"] != false || resp["error"] != "token expired" {
		t.Fatalf("expected token expired, got %+v", resp)
	}
}
