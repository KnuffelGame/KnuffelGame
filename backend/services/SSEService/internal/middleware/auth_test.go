package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/middleware"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

// errorPayload mirrors libs/httpx.ErrorPayload for assertions.
type errorPayload struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func TestAuthMiddleware_MissingJWTCookie(t *testing.T) {
	cfg := testutil.NewTestConfig()
	// BaseURL may be empty; missing cookie is handled first and must yield 401 unauthorized
	log := testutil.NewTestLogger()
	mw := middleware.AuthMiddleware(cfg, log)

	called := false
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)

	if called {
		t.Fatalf("next handler should not be called when jwt cookie is missing")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var got errorPayload
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Error != "unauthorized" || got.Message != "Invalid or expired authentication token" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

func TestAuthMiddleware_InvalidJWT(t *testing.T) {
	// Mock APIGateway /internal/validate returning 401
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/validate" {
			http.NotFound(w, r)
			return
		}
		// ensure Content-Type is json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"error": "invalid",
		})
	}))
	defer gw.Close()

	cfg := testutil.NewTestConfig()
	cfg.APIGatewayBaseURL = gw.URL
	cfg.InternalToken = "int-secret"

	log := testutil.NewTestLogger()
	mw := middleware.AuthMiddleware(cfg, log)

	called := false
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/probe", bytes.NewReader(nil))
	// Add jwt cookie
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)

	if called {
		t.Fatalf("next handler should not be called on invalid JWT")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var got errorPayload
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Error != "unauthorized" || got.Message != "Invalid or expired authentication token" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

func TestAuthMiddleware_ValidJWT_PopulatesContextAndHeaders(t *testing.T) {
	// Mock APIGateway /internal/validate returning 200 with user
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/validate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"valid":    true,
			"user_id":  "usr_ok",
			"username": "Alice",
		})
	}))
	defer gw.Close()

	cfg := testutil.NewTestConfig()
	cfg.APIGatewayBaseURL = gw.URL
	cfg.InternalToken = "int-secret"

	log := testutil.NewTestLogger()
	mw := middleware.AuthMiddleware(cfg, log)

	var gotUserID, gotUsername string
	called := false
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotUserID = r.Header.Get("X-User-ID")
		gotUsername = r.Header.Get("X-Username")
		// Confirm user context accessor works
		uc, ok := middleware.WithUserContext(r)
		if !ok {
			t.Fatalf("WithUserContext returned !ok")
		}
		if uc.UserID != "usr_ok" || uc.Username != "Alice" {
			t.Fatalf("UserContext mismatch: %+v", uc)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	rec := httptest.NewRecorder()
	next.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("next handler not called for valid JWT")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotUserID != "usr_ok" || gotUsername != "Alice" {
		t.Fatalf("headers not set correctly: X-User-ID=%q X-Username=%q", gotUserID, gotUsername)
	}
}
