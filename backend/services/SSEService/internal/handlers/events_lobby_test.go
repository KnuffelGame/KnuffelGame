package handlers_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

type errPayload struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func TestSubscribeLobby_UnauthorizedNoCookie_401(t *testing.T) {
	// Start a router without jwt cookie; AuthMiddleware should reject with 401
	cfg := testutil.NewTestConfig()
	// APIGatewayBaseURL can be non-empty or empty; missing cookie is handled first anyway
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "http://127.0.0.1:9999"))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/events/lobby/lby_abc123")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var got errPayload
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "unauthorized" || got.Message != "Invalid or expired authentication token" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

func TestSubscribeLobby_AuthorizedMember_HeadersAndHeartbeat(t *testing.T) {
	// Mock APIGateway with /internal/validate and /lobbies/{id}
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/validate":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"valid":    true,
				"user_id":  "usr_ok",
				"username": "Alice",
			})
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/lobbies/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"players": []map[string]string{
					{"user_id": "usr_ok"},
				},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(gw.Close)

	cfg := testutil.NewTestConfig()
	cfg.InternalToken = "test-internal" // propagated in validate
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, gw.URL))
	t.Cleanup(srv.Close)

	// Build request with jwt cookie
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/events/lobby/lby_abc123", nil)
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() failed: %v", err)
	}
	defer resp.Body.Close()

	// Assert headers
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Fatalf("Connection = %q, want keep-alive", conn)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("X-Accel-Buffering = %q, want no", xab)
	}

	// Read SSE stream and assert keep_alive within ~300ms (heartbeat interval = 100ms)
	reader := bufio.NewReader(resp.Body)
	foundKeepAlive := false
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read error: %v", err)
		}
		if strings.HasPrefix(line, "event: ") {
			evt := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			if evt == "keep_alive" {
				// Next line should be data: ...
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					t.Fatalf("read data error: %v", err)
				}
				if !strings.HasPrefix(dataLine, "data: ") {
					t.Fatalf("expected data line, got %q", dataLine)
				}
				foundKeepAlive = true
				break
			}
		}
	}
	if !foundKeepAlive {
		t.Fatalf("did not receive keep_alive within 300ms")
	}

	// Disconnect: close body
	_ = resp.Body.Close()
}
