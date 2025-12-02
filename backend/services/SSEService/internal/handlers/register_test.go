package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlers "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/handlers"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/testutil"
)

type errPayloadRegister struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func TestRegisterTarget_CreateTargets_OK(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, "")) // internal endpoints guarded by InternalOnly
	t.Cleanup(srv.Close)

	cases := []struct {
		name       string
		targetType string
		targetID   string
	}{
		{"lobby_target", "lobby", "lby_reg123"},
		{"game_target", "game", "gam_reg456"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := handlers.RegisterTargetRequest{
				TargetType: tc.targetType,
				TargetID:   tc.targetID,
			}
			buf, _ := json.Marshal(body)
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/register", bytes.NewReader(buf))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Internal-Token", cfg.InternalToken)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("POST /internal/register failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			var got handlers.SuccessResponse
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if !got.Success || got.TargetType != tc.targetType || got.TargetID != tc.targetID {
				t.Fatalf("response mismatch: %+v", got)
			}
		})
	}
}

func TestRegisterTarget_ReRegister_Conflict_409(t *testing.T) {
	cfg := testutil.NewTestConfig()
	log := testutil.NewTestLogger()
	srv := httptest.NewServer(testutil.NewTestRouter(cfg, log, ""))
	t.Cleanup(srv.Close)

	target := handlers.RegisterTargetRequest{
		TargetType: "lobby",
		TargetID:   "lby_conflict",
	}
	// First register should succeed
	{
		buf, _ := json.Marshal(target)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/register", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Token", cfg.InternalToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("initial register failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("initial status = %d, want 200", resp.StatusCode)
		}
	}

	// Second register same target should return 409 already_exists
	{
		buf, _ := json.Marshal(target)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/internal/register", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Token", cfg.InternalToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("re-register failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want 409", resp.StatusCode)
		}
		var got errPayloadRegister
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if got.Error != "already_exists" || got.Message != "Target already registered" {
			t.Fatalf("payload mismatch: %+v", got)
		}
	}
}
