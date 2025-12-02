package handlers_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	handlers "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

type unregisterResp struct {
	Success           bool   `json:"success"`
	ConnectionsClosed int    `json:"connections_closed"`
	Message           string `json:"message"`
	Timestamp         string `json:"timestamp"`
}

type errPayloadUnregister struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// TestUnregister_SendsServiceCloseAndClosesConnections validates that:
// - A connected SSE client to a lobby receives a final "service_close" event when /internal/unregister is called.
// - The /internal/unregister response includes connections_closed count (should match active subscribers).
func TestUnregister_SendsServiceCloseAndClosesConnections(t *testing.T) {
	// APIGateway mock: validate OK + lobby membership OK (players includes usr_ok)
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
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, gw.URL))
	t.Cleanup(srv.Close)

	// Open SSE subscription to lobby
	lobbyID := "lby_unreg1"
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/events/lobby/"+lobbyID, nil)
	req.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "jwt-token"})
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer resp.Body.Close()

	// Start reader
	reader := bufio.NewReader(resp.Body)

	// Trigger unregister
	unregBody := handlers.UnregisterTargetRequest{
		TargetType: "lobby",
		TargetID:   lobbyID,
		Reason:     "cleanup",
	}
	payload, _ := json.Marshal(unregBody)
	unregReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/unregister", bytes.NewReader(payload))
	unregReq.Header.Set("Content-Type", "application/json")
	unregReq.Header.Set("X-Internal-Token", cfg.InternalToken)
	unregResp, err := http.DefaultClient.Do(unregReq)
	if err != nil {
		t.Fatalf("unregister request failed: %v", err)
	}
	defer unregResp.Body.Close()

	if unregResp.StatusCode != http.StatusOK {
		t.Fatalf("unregister status = %d, want 200", unregResp.StatusCode)
	}
	var unregJSON unregisterResp
	if err := json.NewDecoder(unregResp.Body).Decode(&unregJSON); err != nil {
		t.Fatalf("decode unregister response failed: %v", err)
	}
	if !unregJSON.Success || unregJSON.ConnectionsClosed < 1 {
		t.Fatalf("unexpected unregister response: %+v", unregJSON)
	}

	// Expect service_close event on the SSE stream
	foundClose := false
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break // server may close after sending event; loop ends
		}
		if strings.HasPrefix(line, "event: ") {
			ev := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			if ev == "service_close" {
				// Next line should be data: ...
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					t.Fatalf("read data line error: %v", err)
				}
				if !strings.HasPrefix(dataLine, "data: ") {
					t.Fatalf("expected data line, got %q", dataLine)
				}
				foundClose = true
				break
			}
		}
	}
	if !foundClose {
		t.Fatalf("did not receive service_close event after unregister")
	}
}
