package integration_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	handlers "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

// waitForEvent scans an SSE stream until the given event name is observed
// or until deadline. It ignores other events (e.g., keep_alive).
func waitForEvent(t *testing.T, r *bufio.Reader, event string, timeout time.Duration) (bool, string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return false, ""
			}
			// allow brief retry on transient read issues within timeout window
			if time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return false, ""
		}
		// Skip comments/empty and retry directive lines
		if line == "\n" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "retry: ") {
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			ev := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
			// capture data line if present
			var dataLine string
			if dl, err := r.ReadString('\n'); err == nil && strings.HasPrefix(dl, "data: ") {
				dataLine = strings.TrimSpace(strings.TrimPrefix(dl, "data: "))
			}
			if ev == event {
				return true, dataLine
			}
			// otherwise continue scanning
		}
	}
	return false, ""
}

// waitNoEvent ensures the given event name is NOT observed within timeout (ignoring other events like keep_alive).
func waitNoEvent(t *testing.T, r *bufio.Reader, event string, timeout time.Duration) bool {
	t.Helper()
	found, _ := waitForEvent(t, r, event, timeout)
	return !found
}

func openSSE(t *testing.T, url string, jwtValue string, timeout time.Duration) (*http.Response, *bufio.Reader) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtValue})
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open SSE failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("SSE status = %d, want 200", resp.StatusCode)
	}
	return resp, bufio.NewReader(resp.Body)
}

// TestIntegration_Lobby_MultiClient_BroadcastAndTargeted_Unregister
// - Two clients subscribe to the same lobby
// - Publish broadcast event: both receive
// - Publish targeted event: only one receives
// - Unregister: both receive "service_close", stats show target removed
func TestIntegration_Lobby_MultiClient_BroadcastAndTargeted_Unregister(t *testing.T) {
	// Mock APIGateway: /internal/validate decodes {"token":...}, /lobbies/{id} membership ok
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/validate":
			w.Header().Set("Content-Type", "application/json")
			var vb struct {
				Token string `json:"token"`
			}
			_ = json.NewDecoder(r.Body).Decode(&vb)
			var uid, uname string
			switch vb.Token {
			case "t1":
				uid, uname = "usr1", "U1"
			case "t2":
				uid, uname = "usr2", "U2"
			default:
				uid, uname = "usrX", "UX"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"valid":    true,
				"user_id":  uid,
				"username": uname,
			})
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/lobbies/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"players": []map[string]string{
					{"user_id": "usr1"},
					{"user_id": "usr2"},
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
	// Keep short heartbeats to follow project testing convention; we filter out heartbeats explicitly.
	cfg.HeartbeatInterval = 100 * time.Millisecond
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, gw.URL))
	t.Cleanup(srv.Close)

	lobbyID := "lby_integ1"

	// Open two clients
	resp1, r1 := openSSE(t, srv.URL+"/events/lobby/"+lobbyID, "t1", 2*time.Second)
	defer resp1.Body.Close()
	resp2, r2 := openSSE(t, srv.URL+"/events/lobby/"+lobbyID, "t2", 2*time.Second)
	defer resp2.Body.Close()

	// Publish a broadcast event
	pubBody := handlers.PublishEventRequest{
		TargetType: "lobby",
		TargetID:   lobbyID,
		EventType:  "player_joined",
		Data:       map[string]any{"user_id": "usr_new"},
	}
	pubBuf, _ := json.Marshal(pubBody)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/publish", bytes.NewReader(pubBuf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", cfg.InternalToken)
	if resp, err := http.DefaultClient.Do(req); err != nil {
		t.Fatalf("publish broadcast failed: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("publish status = %d, want 200", resp.StatusCode)
		}
	}

	// Both clients should receive event: player_joined
	if ok, _ := waitForEvent(t, r1, "player_joined", 500*time.Millisecond); !ok {
		t.Fatalf("client1 did not receive broadcast event")
	}
	if ok, _ := waitForEvent(t, r2, "player_joined", 500*time.Millisecond); !ok {
		t.Fatalf("client2 did not receive broadcast event")
	}

	// Targeted publish to usr2 only
	pubBodyTargeted := handlers.PublishEventRequest{
		TargetType:   "lobby",
		TargetID:     lobbyID,
		EventType:    "player_active",
		TargetUserID: "usr2",
		Data:         map[string]any{"active": true},
	}
	pubTBuf, _ := json.Marshal(pubBodyTargeted)
	reqT, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/publish", bytes.NewReader(pubTBuf))
	reqT.Header.Set("Content-Type", "application/json")
	reqT.Header.Set("X-Internal-Token", cfg.InternalToken)
	if resp, err := http.DefaultClient.Do(reqT); err != nil {
		t.Fatalf("publish targeted failed: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("publish targeted status = %d, want 200", resp.StatusCode)
		}
	}

	// Client 2 should receive targeted event, client 1 should not within a small window
	if ok, _ := waitForEvent(t, r2, "player_active", 500*time.Millisecond); !ok {
		t.Fatalf("client2 did not receive targeted event")
	}
	if keep := waitNoEvent(t, r1, "player_active", 300*time.Millisecond); !keep {
		t.Fatalf("client1 unexpectedly received targeted event")
	}

	// Unregister lobby - clients should receive service_close
	unreg := handlers.UnregisterTargetRequest{
		TargetType: "lobby",
		TargetID:   lobbyID,
		Reason:     "cleanup",
	}
	unregBuf, _ := json.Marshal(unreg)
	ureq, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/unregister", bytes.NewReader(unregBuf))
	ureq.Header.Set("Content-Type", "application/json")
	ureq.Header.Set("X-Internal-Token", cfg.InternalToken)
	uresp, err := http.DefaultClient.Do(ureq)
	if err != nil {
		t.Fatalf("unregister request failed: %v", err)
	}
	defer uresp.Body.Close()
	if uresp.StatusCode != http.StatusOK {
		t.Fatalf("unregister status = %d, want 200", uresp.StatusCode)
	}

	// Both clients should receive service_close
	if ok, _ := waitForEvent(t, r1, "service_close", 500*time.Millisecond); !ok {
		t.Fatalf("client1 did not receive service_close")
	}
	if ok, _ := waitForEvent(t, r2, "service_close", 500*time.Millisecond); !ok {
		t.Fatalf("client2 did not receive service_close")
	}

	// Stats should show 0 targets after a brief moment
	time.Sleep(50 * time.Millisecond)
	statsReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/connections", nil)
	statsReq.Header.Set("X-Internal-Token", cfg.InternalToken)
	statsResp, err := http.DefaultClient.Do(statsReq)
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	defer statsResp.Body.Close()
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, want 200", statsResp.StatusCode)
	}
	var stats struct {
		TotalTargets     int `json:"total_targets"`
		TotalConnections int `json:"total_connections"`
	}
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats failed: %v", err)
	}
	if stats.TotalTargets != 0 || stats.TotalConnections != 0 {
		t.Fatalf("stats after unregister mismatch: %+v", stats)
	}
}

// TestIntegration_Reconnect_HeartbeatResumes
// - Client connects, closes, and reconnects
// - After reconnect, keep_alive events resume (we assert receiving within 300ms)
func TestIntegration_Reconnect_HeartbeatResumes(t *testing.T) {
	// Mock APIGateway validate + lobby membership ok
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/validate":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"valid":    true,
				"user_id":  "usrR",
				"username": "Reconn",
			})
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/lobbies/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"players": []map[string]string{
					{"user_id": "usrR"},
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
	cfg.HeartbeatInterval = 100 * time.Millisecond
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, gw.URL))
	t.Cleanup(srv.Close)

	lobbyID := "lby_reconn"

	// First connection
	resp, rdr := openSSE(t, srv.URL+"/events/lobby/"+lobbyID, "tok", 2*time.Second)
	// Receive a keep_alive within 300ms
	if ok, _ := waitForEvent(t, rdr, "keep_alive", 300*time.Millisecond); !ok {
		t.Fatalf("keep_alive not received on initial connect")
	}
	_ = resp.Body.Close()

	// Reconnect
	resp2, rdr2 := openSSE(t, srv.URL+"/events/lobby/"+lobbyID, "tok", 2*time.Second)
	defer resp2.Body.Close()
	if ok, _ := waitForEvent(t, rdr2, "keep_alive", 300*time.Millisecond); !ok {
		t.Fatalf("keep_alive not received after reconnect")
	}
}
