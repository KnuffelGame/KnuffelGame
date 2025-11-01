package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/joincode"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use environment variables or defaults for test database
	// In a real environment, this would connect to a test database
	connStr := "host=localhost port=5432 user=lobby password=secure dbname=lobby_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("skipping test: cannot connect to test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping test: cannot ping test database: %v", err)
	}
	return db
}

func TestCreateLobby_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(db, codeGen)

	userID := uuid.New()
	username := "TestUser"

	req := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.CreateLobbyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate response
	if resp.LobbyID == uuid.Nil {
		t.Error("expected non-nil lobby_id")
	}
	if resp.JoinCode == "" {
		t.Error("expected non-empty join_code")
	}
	if len(resp.JoinCode) != 6 {
		t.Errorf("expected join_code length 6, got %d", len(resp.JoinCode))
	}
	if resp.LeaderID != userID {
		t.Errorf("expected leader_id %s, got %s", userID, resp.LeaderID)
	}
	if resp.Status != models.LobbyStatusWaiting {
		t.Errorf("expected status %s, got %s", models.LobbyStatusWaiting, resp.Status)
	}
	if len(resp.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(resp.Players))
	}
	if len(resp.Players) > 0 {
		player := resp.Players[0]
		if player.UserID != userID {
			t.Errorf("expected player user_id %s, got %s", userID, player.UserID)
		}
		if player.Username != username {
			t.Errorf("expected player username %s, got %s", username, player.Username)
		}
		if player.JoinedAt.IsZero() {
			t.Error("expected joined_at to be set")
		}
		if time.Since(player.JoinedAt) > time.Minute {
			t.Errorf("joined_at timestamp seems too old: %v", player.JoinedAt)
		}
		if !player.IsActive {
			t.Error("expected player to be active")
		}
	}

	// Cleanup
	db.Exec("DELETE FROM players WHERE lobby_id = $1", resp.LobbyID)
	db.Exec("DELETE FROM lobbies WHERE id = $1", resp.LobbyID)
	db.Exec("DELETE FROM users WHERE id = $1", userID)
}

func TestCreateLobby_MissingUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(db, codeGen)

	req := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req.Header.Set(headerUsername, "TestUser")
	// Missing X-User-ID header

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLobby_MissingUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(db, codeGen)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req.Header.Set(headerUserID, userID.String())
	// Missing X-Username header

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLobby_InvalidUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(db, codeGen)

	req := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req.Header.Set(headerUserID, "not-a-uuid")
	req.Header.Set(headerUsername, "TestUser")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateLobby_IdempotentUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(db, codeGen)

	userID := uuid.New()
	username := "TestUser"

	// Create first lobby
	req1 := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req1.Header.Set(headerUserID, userID.String())
	req1.Header.Set(headerUsername, username)

	rec1 := httptest.NewRecorder()
	h(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201, got %d", rec1.Code)
	}

	var resp1 models.CreateLobbyResponse
	json.Unmarshal(rec1.Body.Bytes(), &resp1)

	// Create second lobby with same user (should succeed - user already exists)
	req2 := httptest.NewRequest(http.MethodPost, "/lobbies", nil)
	req2.Header.Set(headerUserID, userID.String())
	req2.Header.Set(headerUsername, username)

	rec2 := httptest.NewRecorder()
	h(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("second request: expected 201, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp2 models.CreateLobbyResponse
	json.Unmarshal(rec2.Body.Bytes(), &resp2)

	// Validate both lobbies are different but same user
	if resp1.LobbyID == resp2.LobbyID {
		t.Error("expected different lobby IDs")
	}
	if resp1.JoinCode == resp2.JoinCode {
		t.Error("expected different join codes")
	}
	if resp1.LeaderID != resp2.LeaderID {
		t.Error("expected same leader ID")
	}

	// Cleanup
	db.Exec("DELETE FROM players WHERE lobby_id = $1", resp1.LobbyID)
	db.Exec("DELETE FROM lobbies WHERE id = $1", resp1.LobbyID)
	db.Exec("DELETE FROM players WHERE lobby_id = $1", resp2.LobbyID)
	db.Exec("DELETE FROM lobbies WHERE id = $1", resp2.LobbyID)
	db.Exec("DELETE FROM users WHERE id = $1", userID)
}
