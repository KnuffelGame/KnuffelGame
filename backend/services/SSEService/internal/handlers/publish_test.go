package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	handlers "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

type publishResp struct {
	Success           bool `json:"success"`
	ConnectionsFound  int  `json:"connections_found"`
	EventsSent        int  `json:"events_sent"`
	FailedConnections int  `json:"failed_connections"`
}

type errPayloadPublish struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// TestPublish_TargetNotFound_404 verifies 404 and error payload when target entry is missing.
func TestPublish_TargetNotFound_404(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	body := handlers.PublishEventRequest{
		TargetType: "lobby",
		TargetID:   "lby_missing",
		EventType:  "player_joined",
		Data:       map[string]interface{}{"x": 1},
	}
	buf, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/publish", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", cfg.InternalToken)

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /internal/publish failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var got errPayloadPublish
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "target_not_found" || got.Message != "Target entry not found" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

// TestPublish_TargetFound_Counters_OK registers a target then publishes, asserting counters present.
func TestPublish_TargetFound_Counters_OK(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	// Register target first
	regReq := handlers.RegisterTargetRequest{TargetType: "game", TargetID: "gam_pub1"}
	regBuf, _ := json.Marshal(regReq)
	regHTTPReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/register", bytes.NewReader(regBuf))
	regHTTPReq.Header.Set("Content-Type", "application/json")
	regHTTPReq.Header.Set("X-Internal-Token", cfg.InternalToken)
	if regResp, err := http.DefaultClient.Do(regHTTPReq); err != nil {
		t.Fatalf("register failed: %v", err)
	} else {
		defer regResp.Body.Close()
		if regResp.StatusCode != http.StatusOK {
			t.Fatalf("register status = %d, want 200", regResp.StatusCode)
		}
	}

	// Publish to existing target (no active subscribers in unit test)
	body := handlers.PublishEventRequest{
		TargetType: "game",
		TargetID:   "gam_pub1",
		EventType:  "turn_changed",
		Data:       map[string]interface{}{"turn": 2},
	}
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/publish", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", cfg.InternalToken)

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got publishResp
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	// With no subscribers, counters should be zero but Success true.
	if !got.Success {
		t.Fatalf("Success = false, want true")
	}
	if got.ConnectionsFound != 0 || got.EventsSent != 0 || got.FailedConnections != 0 {
		t.Fatalf("counters mismatch: %+v (expected all zeros)", got)
	}
}

// TestPublish_TargetedUser_Counters_OK publishes with target_user_id; counters should be valid envelope.
func TestPublish_TargetedUser_Counters_OK(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	// Register target first
	regReq := handlers.RegisterTargetRequest{TargetType: "lobby", TargetID: "lby_pub_targeted"}
	regBuf, _ := json.Marshal(regReq)
	regHTTPReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/register", bytes.NewReader(regBuf))
	regHTTPReq.Header.Set("Content-Type", "application/json")
	regHTTPReq.Header.Set("X-Internal-Token", cfg.InternalToken)
	if regResp, err := http.DefaultClient.Do(regHTTPReq); err != nil {
		t.Fatalf("register failed: %v", err)
	} else {
		defer regResp.Body.Close()
		if regResp.StatusCode != http.StatusOK {
			t.Fatalf("register status = %d, want 200", regResp.StatusCode)
		}
	}

	// Targeted publish (no subscribers in unit test; verify envelope fields)
	body := handlers.PublishEventRequest{
		TargetType:   "lobby",
		TargetID:     "lby_pub_targeted",
		EventType:    "player_active",
		TargetUserID: "usr_target",
		Data:         map[string]interface{}{"active": true},
	}
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/publish", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", cfg.InternalToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got publishResp
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !got.Success {
		t.Fatalf("Success = false, want true")
	}
	// No subscribers here; counters remain zero.
	if got.ConnectionsFound != 0 || got.EventsSent != 0 || got.FailedConnections != 0 {
		t.Fatalf("counters mismatch: %+v (expected all zeros)", got)
	}
}
