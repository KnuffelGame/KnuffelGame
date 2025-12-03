package handlers_test

import (
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

// TestPublish_LobbyNotFound_404 verifies 404 when publishing to lobby with no active connections.
func TestPublish_LobbyNotFound_404(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	body := handlers.PublishEventRequest{
		LobbyID:   "lby_missing",
		EventType: "player_joined",
		Data:      map[string]interface{}{"x": 1},
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
	if got.Error != "target_not_found" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

// TestPublish_Validation_KeepAliveRejected verifies event_type "keep_alive" is rejected with 400.
func TestPublish_Validation_KeepAliveRejected(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	body := handlers.PublishEventRequest{
		LobbyID:   "lby_test",
		EventType: "keep_alive", // reserved
		Data:      map[string]interface{}{"timestamp": 1234567890000},
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var got errPayloadPublish
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "invalid_request" || !strings.Contains(got.Message, "keep_alive") {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

// TestPublish_Validation_EventTypeTooLong verifies event_type > 128 chars is rejected.
func TestPublish_Validation_EventTypeTooLong(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	longEventType := strings.Repeat("a", 129) // 129 chars
	body := handlers.PublishEventRequest{
		LobbyID:   "lby_test",
		EventType: longEventType,
		Data:      map[string]interface{}{"test": "data"},
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var got errPayloadPublish
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "invalid_request" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

// TestPublish_Validation_EventTypeEmpty verifies empty event_type is rejected.
func TestPublish_Validation_EventTypeEmpty(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	body := handlers.PublishEventRequest{
		LobbyID:   "lby_test",
		EventType: "", // empty
		Data:      map[string]interface{}{"test": "data"},
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var got errPayloadPublish
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "invalid_request" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}

// TestPublish_Validation_MissingLobbyID verifies missing lobby_id is rejected.
func TestPublish_Validation_MissingLobbyID(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal guard active
	t.Cleanup(srv.Close)

	body := handlers.PublishEventRequest{
		LobbyID:   "", // missing
		EventType: "player_joined",
		Data:      map[string]interface{}{"test": "data"},
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var got errPayloadPublish
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got.Error != "invalid_request" {
		t.Fatalf("payload mismatch: %+v", got)
	}
}
