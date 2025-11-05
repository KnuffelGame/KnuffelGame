package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Note: Tests previously used a real test database. Refactored to use sqlmock.

func TestGetLobby_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	// Create test user
	userID := uuid.New()
	username := "Alice"

	// Create test lobby and player
	lobbyID := uuid.New()
	joinCode := "ABC123"
	playerID := uuid.New()
	joinedAt := time.Now()

	// Expect query and return one row
	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).AddRow(lobbyID.String(), joinCode, models.LobbyStatusWaiting, userID.String(), playerID.String(), userID.String(), username, joinedAt, true)
	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(rows)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Username", username)

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response models.LobbyDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify lobby details
	if response.LobbyID != lobbyID {
		t.Errorf("expected lobby_id %s, got %s", lobbyID, response.LobbyID)
	}
	if response.JoinCode != joinCode {
		t.Errorf("expected join_code %s, got %s", joinCode, response.JoinCode)
	}
	if response.Status != models.LobbyStatusWaiting {
		t.Errorf("expected status %s, got %s", models.LobbyStatusWaiting, response.Status)
	}
	if response.LeaderID != userID {
		t.Errorf("expected leader_id %s, got %s", userID, response.LeaderID)
	}

	// Verify players
	if len(response.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(response.Players))
	}
	player := response.Players[0]
	if player.UserID != userID {
		t.Errorf("expected player user_id %s, got %s", userID, player.UserID)
	}
	if player.Username != username {
		t.Errorf("expected player username %s, got %s", username, player.Username)
	}
	if !player.IsActive {
		t.Errorf("expected player to be active")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobby_MultiplePlayersLeaderIdentified(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	leaderID := uuid.New()
	leaderName := "Alice"
	player2ID := uuid.New()
	player2Name := "Bob"
	player3ID := uuid.New()
	player3Name := "Charlie"

	lobbyID := uuid.New()
	joinCode := "XYZ789"
	joinedAt1 := time.Now().Add(-3 * time.Minute)
	joinedAt2 := time.Now().Add(-2 * time.Minute)
	joinedAt3 := time.Now().Add(-1 * time.Minute)

	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).
		AddRow(lobbyID.String(), joinCode, models.LobbyStatusWaiting, leaderID.String(), uuid.New().String(), leaderID.String(), leaderName, joinedAt1, true).
		AddRow(lobbyID.String(), joinCode, models.LobbyStatusWaiting, leaderID.String(), uuid.New().String(), player2ID.String(), player2Name, joinedAt2, true).
		AddRow(lobbyID.String(), joinCode, models.LobbyStatusWaiting, leaderID.String(), uuid.New().String(), player3ID.String(), player3Name, joinedAt3, true)

	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(rows)

	// Create request as player2
	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)
	req.Header.Set("X-User-ID", player2ID.String())
	req.Header.Set("X-Username", player2Name)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response models.LobbyDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.LeaderID != leaderID {
		t.Errorf("expected leader_id %s, got %s", leaderID, response.LeaderID)
	}

	if len(response.Players) != 3 {
		t.Fatalf("expected 3 players, got %d", len(response.Players))
	}

	expectedUserIDs := []uuid.UUID{leaderID, player2ID, player3ID}
	for i, player := range response.Players {
		if player.UserID != expectedUserIDs[i] {
			t.Errorf("player %d: expected user_id %s, got %s", i, expectedUserIDs[i], player.UserID)
		}
		if !player.IsActive {
			t.Errorf("player %d: expected to be active", i)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobby_WithInactivePlayers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	leaderID := uuid.New()
	inactivePlayerID := uuid.New()
	lobbyID := uuid.New()
	joinCode := "TEST99"
	joinedAt1 := time.Now().Add(-10 * time.Minute)
	joinedAt2 := time.Now().Add(-5 * time.Minute)

	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).
		AddRow(lobbyID.String(), joinCode, models.LobbyStatusInGame, leaderID.String(), uuid.New().String(), leaderID.String(), "Leader", joinedAt1, true).
		AddRow(lobbyID.String(), joinCode, models.LobbyStatusInGame, leaderID.String(), uuid.New().String(), inactivePlayerID.String(), "InactivePlayer", joinedAt2, false)

	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)
	req.Header.Set("X-User-ID", leaderID.String())
	req.Header.Set("X-Username", "Leader")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response models.LobbyDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(response.Players))
	}

	activeCount := 0
	inactiveCount := 0
	for _, player := range response.Players {
		if player.IsActive {
			activeCount++
			if player.UserID != leaderID {
				t.Errorf("active player should be leader, got %s", player.UserID)
			}
		} else {
			inactiveCount++
			if player.UserID != inactivePlayerID {
				t.Errorf("inactive player should be %s, got %s", inactivePlayerID, player.UserID)
			}
		}
	}

	if activeCount != 1 {
		t.Errorf("expected 1 active player, got %d", activeCount)
	}
	if inactiveCount != 1 {
		t.Errorf("expected 1 inactive player, got %d", inactiveCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobby_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "Alice"
	nonExistentLobbyID := uuid.New()

	// Expect query but return no rows
	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	mock.ExpectQuery("SELECT").WithArgs(nonExistentLobbyID.String()).WillReturnRows(sqlmock.NewRows(columns))

	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+nonExistentLobbyID.String(), nil)
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Username", username)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", nonExistentLobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobby_MissingHeaders(t *testing.T) {
	// No DB interactions expected
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()

	// Create request without headers
	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLobby_InvalidLobbyID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "Alice"

	req := httptest.NewRequest(http.MethodGet, "/lobbies/invalid-uuid", nil)
	req.Header.Set("X-User-ID", userID.String())
	req.Header.Set("X-Username", username)

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLobby_InvalidUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)
	req.Header.Set("X-User-ID", "invalid-uuid")
	req.Header.Set("X-Username", "Alice")

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLobby_ForbiddenForNonMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	memberID := uuid.New()
	nonMemberID := uuid.New()
	lobbyID := uuid.New()

	// Return rows showing only member is in lobby
	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).
		AddRow(lobbyID.String(), "FORBID", models.LobbyStatusWaiting, memberID.String(), uuid.New().String(), memberID.String(), "Member", time.Now(), true)

	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/lobbies/"+lobbyID.String(), nil)
	req.Header.Set("X-User-ID", nonMemberID.String())
	req.Header.Set("X-Username", "Stranger")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyHandler(db)
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}
