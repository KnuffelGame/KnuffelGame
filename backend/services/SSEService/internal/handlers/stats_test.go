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

type statsResp struct {
	Timestamp        string `json:"timestamp"`
	TotalTargets     int    `json:"total_targets"`
	TotalConnections int    `json:"total_connections"`
	LobbyTargets     int    `json:"lobby_targets"`
	LobbyConnections int    `json:"lobby_connections"`
	GameTargets      int    `json:"game_targets"`
	GameConnections  int    `json:"game_connections"`
}

// TestGetConnectionStats_ReturnsAggregates verifies /internal/connections returns correct totals.
// Scenario: 2 lobby subscribers (usr1, usr2) to same lobby, 1 game subscriber (usr3).
func TestGetConnectionStats_ReturnsAggregates(t *testing.T) {
	// APIGateway mock with validate returning distinct users based on cookie token,
	// and lobby membership including usr1 & usr2; game returns ok.
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/validate":
			w.Header().Set("Content-Type", "application/json")
			// Auth middleware posts {"token": cookieValue}; decode body to select user
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
			case "t3":
				uid, uname = "usr3", "U3"
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
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/games/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(gw.Close)

	cfg := testutil.NewTestConfig()
	cfg.HeartbeatInterval = 10 * time.Second
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, gw.URL))
	t.Cleanup(srv.Close)

	// Open two lobby subscriptions for the same lobby with different users
	lobbyID := "lby_stats"
	req1, _ := http.NewRequest(http.MethodGet, srv.URL+"/events/lobby/"+lobbyID, nil)
	req1.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "t1"})
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("subscribe 1 failed: %v", err)
	}
	t.Cleanup(func() { resp1.Body.Close() })

	reader1 := bufio.NewReader(resp1.Body)
	// Read initial lines quickly to ensure connection established
	_, _ = reader1.ReadString('\n')

	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/events/lobby/"+lobbyID, nil)
	req2.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "t2"})
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("subscribe 2 failed: %v", err)
	}
	t.Cleanup(func() { resp2.Body.Close() })
	reader2 := bufio.NewReader(resp2.Body)
	_, _ = reader2.ReadString('\n')

	// Open one game subscription
	gameID := "gam_stats"
	req3, _ := http.NewRequest(http.MethodGet, srv.URL+"/events/game/"+gameID, nil)
	req3.AddCookie(&http.Cookie{Name: cfg.JWTCookieName, Value: "t3"})
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("subscribe game failed: %v", err)
	}
	t.Cleanup(func() { resp3.Body.Close() })
	reader3 := bufio.NewReader(resp3.Body)
	_, _ = reader3.ReadString('\n')

	// Small wait bounded & under heartbeat to allow registry updates
	time.Sleep(50 * time.Millisecond)

	// Fetch stats
	statsReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/internal/connections", nil)
	statsReq.Header.Set("X-Internal-Token", cfg.InternalToken)
	statsHTTPResp, err := http.DefaultClient.Do(statsReq)
	if err != nil {
		t.Fatalf("GET /internal/connections failed: %v", err)
	}
	defer statsHTTPResp.Body.Close()

	if statsHTTPResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", statsHTTPResp.StatusCode)
	}
	var got statsResp
	if err := json.NewDecoder(statsHTTPResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	// Assert aggregates: 2 targets (lobby + game), 3 connections, split accordingly
	if got.TotalTargets != 2 || got.TotalConnections != 3 ||
		got.LobbyTargets != 1 || got.LobbyConnections != 2 ||
		got.GameTargets != 1 || got.GameConnections != 1 {
		t.Fatalf("stats mismatch: %+v", got)
	}
}
