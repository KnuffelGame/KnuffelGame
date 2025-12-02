package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewTestConfig_Values verifies default short heartbeat and jwt cookie name.
func TestNewTestConfig_Values(t *testing.T) {
	cfg := NewTestConfig()
	if cfg == nil {
		t.Fatal("NewTestConfig returned nil")
	}
	if cfg.HeartbeatInterval != 100*time.Millisecond {
		t.Fatalf("HeartbeatInterval = %v, want 100ms", cfg.HeartbeatInterval)
	}
	if cfg.JWTCookieName != "jwt" {
		t.Fatalf("JWTCookieName = %q, want jwt", cfg.JWTCookieName)
	}
	if cfg.InternalToken != "test-internal" {
		t.Fatalf("InternalToken = %q, want test-internal", cfg.InternalToken)
	}
	if cfg.RateLimitSSEPerMinute != 0 {
		t.Fatalf("RateLimitSSEPerMinute = %d, want 0", cfg.RateLimitSSEPerMinute)
	}
}

// TestNewTestRouter_APIGatewayInjection ensures we can inject a mocked APIGateway BaseURL into cfg.
func TestNewTestRouter_APIGatewayInjection(t *testing.T) {
	cfg := NewTestConfig()
	log := NewTestLogger()

	// Mock gateway URL (not used in this test; just verify injection)
	mockBase := "http://127.0.0.1:9999"
	h := NewTestRouter(cfg, log, mockBase)
	if h == nil {
		t.Fatal("NewTestRouter returned nil")
	}
	if cfg.APIGatewayBaseURL != mockBase {
		t.Fatalf("APIGatewayBaseURL = %q, want %q", cfg.APIGatewayBaseURL, mockBase)
	}
}

// TestNewTestRouter_HealthcheckMounted verifies the router mounts /healthcheck and returns JSON.
func TestNewTestRouter_HealthcheckMounted(t *testing.T) {
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
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "{\"status\":\"ok\"}\n" && string(body) != "{\"status\":\"ok\"}" {
		t.Fatalf("body = %q, want {\"status\":\"ok\"}", string(body))
	}
}
