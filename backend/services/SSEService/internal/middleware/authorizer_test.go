package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/middleware"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
	"github.com/go-chi/chi/v5"
)

type respErr struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, val)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rc)
	return r.WithContext(ctx)
}

func TestAuthorizeLobbyMembership_Member_OK(t *testing.T) {
	// Mock APIGateway /lobbies/{id} returning 200 with players including user
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/lobbies/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"players": []map[string]string{
				{"user_id": "usr_ok"},
			},
		})
	}))
	defer gw.Close()

	cfg := testutil.NewTestConfig()
	cfg.LobbyServiceBaseURL = gw.URL
	log := testutil.NewTestLogger()
	mw := middleware.AuthorizeLobbyMembership(cfg, log)

	called := false
	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/events/lobby/lby_abc123", nil)
	req = withChiParam(req, "lobby_id", "lby_abc123")
	req.Header.Set("X-User-ID", "usr_ok")
	req.Header.Set("X-Username", "Alice")
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("next handler was not called for authorized lobby membership")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAuthorizeLobbyMembership_NotMember_403(t *testing.T) {
	// Mock APIGateway returns 200 with players excluding user (middleware enforces membership)
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/lobbies/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"players": []map[string]string{
				{"user_id": "usr_other"},
			},
		})
	}))
	defer gw.Close()

	cfg := testutil.NewTestConfig()
	cfg.LobbyServiceBaseURL = gw.URL
	log := testutil.NewTestLogger()
	mw := middleware.AuthorizeLobbyMembership(cfg, log)

	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/events/lobby/lby_abc123", nil)
	req = withChiParam(req, "lobby_id", "lby_abc123")
	req.Header.Set("X-User-ID", "usr_nope")
	req.Header.Set("X-Username", "Alice")
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	var got respErr
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Error != "forbidden" || got.Message != "You are not a member of this lobby" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

func TestAuthorizeLobbyMembership_NotFound_404(t *testing.T) {
	// Mock APIGateway returns 404
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"lobby_not_found","message":"Lobby not found"}`))
	}))
	defer gw.Close()

	cfg := testutil.NewTestConfig()
	cfg.LobbyServiceBaseURL = gw.URL
	log := testutil.NewTestLogger()
	mw := middleware.AuthorizeLobbyMembership(cfg, log)

	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/events/lobby/lby_abc123", nil)
	req = withChiParam(req, "lobby_id", "lby_abc123")
	req.Header.Set("X-User-ID", "usr_ok")
	req.Header.Set("X-Username", "Alice")
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var got respErr
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// Current implementation returns error "not_found" via httpx.WriteNotFound; message must match.
	if got.Message != "Lobby not found" {
		t.Fatalf("message mismatch: %+v", got)
	}
}

func TestAuthorizeLobbyMembership_LobbyServiceBaseURLMissing_500(t *testing.T) {
	// When LobbyServiceBaseURL is missing, middleware must return 500 InternalServerError.
	cfg := testutil.NewTestConfig()
	cfg.LobbyServiceBaseURL = ""
	log := testutil.NewTestLogger()
	mw := middleware.AuthorizeLobbyMembership(cfg, log)

	next := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/events/lobby/lby_abc123", nil)
	req = withChiParam(req, "lobby_id", "lby_abc123")
	req.Header.Set("X-User-ID", "usr_ok")
	req.Header.Set("X-Username", "Alice")
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var got respErr
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Error != "internal_error" || got.Message != "Internal server error" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}
