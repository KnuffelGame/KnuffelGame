package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/services/AuthService/internal/jwt"
)

func TestCreateTokenHandler_Success(t *testing.T) {
	gen := jwt.NewGenerator("secret")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]string{"username": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["token"] == "" {
		t.Fatalf("expected token field in response, got %+v", resp)
	}
}

func TestCreateTokenHandler_ValidationFail(t *testing.T) {
	gen := jwt.NewGenerator("secret")
	h := CreateTokenHandler(gen)
	body, _ := json.Marshal(map[string]string{"username": "Al"}) // too short
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
