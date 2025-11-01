package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
)

func TestCreateToken_Success(t *testing.T) {
	gen := jwt.NewGenerator("12345678901234567890123456789012")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]interface{}{"username": "Alice", "user_id": "550e8400-e29b-41d4-a716-446655440000"})
	req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if _, ok := resp["token"].(string); !ok || resp["token"].(string) == "" {
		t.Fatalf("expected token field, got %+v", resp)
	}
}

func TestCreateToken_MissingUserID(t *testing.T) {
	gen := jwt.NewGenerator("12345678901234567890123456789012")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]interface{}{"username": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateToken_ValidationFail(t *testing.T) {
	gen := jwt.NewGenerator("12345678901234567890123456789012")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]interface{}{"username": "Al", "user_id": "550e8400-e29b-41d4-a716-446655440000"}) // username too short
	req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateToken_UnknownField(t *testing.T) {
	gen := jwt.NewGenerator("12345678901234567890123456789012")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]interface{}{"username": "Alice", "user_id": "550e8400-e29b-41d4-a716-446655440000", "extra": "x"})
	req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid_request" {
		t.Fatalf("expected invalid_request error, got %+v", resp)
	}
}
