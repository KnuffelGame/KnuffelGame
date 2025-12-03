package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	router "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

func TestRouter_Healthcheck(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	r := router.New(cfg, log)

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthcheck => status %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestRouter_PublicRoutes(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	r := router.New(cfg, log)

	// Test SSE lobby route exists (should return 401 due to missing auth)
	req := httptest.NewRequest(http.MethodGet, "/events/lobby/550e8400-e29b-41d4-a716-446655440000", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Should get 401 unauthorized (auth middleware)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /events/lobby/{id} => status %d, want 401", rec.Code)
	}
}

func TestRouter_InternalRoutes(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	r := router.New(cfg, log)

	// Test publish route exists (should return 400 due to missing token or invalid request)
	req := httptest.NewRequest(http.MethodPost, "/internal/publish", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Should get some error status (internal guard or validation)
	if rec.Code == http.StatusOK {
		t.Fatalf("POST /internal/publish => status %d, expected error status", rec.Code)
	}
}

func TestRouter_UnknownRoutes(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	r := router.New(cfg, log)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /unknown => status %d, want 404", rec.Code)
	}
}
