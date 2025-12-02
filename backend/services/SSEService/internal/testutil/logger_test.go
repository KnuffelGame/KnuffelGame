package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewTestLogger_NotNil ensures the test logger is constructed.
func TestNewTestLogger_NotNil(t *testing.T) {
	log := NewTestLogger()
	if log == nil {
		t.Fatal("NewTestLogger returned nil")
	}
}

// TestNewTestRouter_HealthcheckWithLogger verifies router works with the test logger.
func TestNewTestRouter_HealthcheckWithLogger(t *testing.T) {
	cfg := NewTestConfig()
	log := NewTestLogger()
	h := NewTestRouter(cfg, log, "")

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthcheck => status %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}
