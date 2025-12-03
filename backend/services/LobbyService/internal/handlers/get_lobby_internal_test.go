package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestGetLobbyInternal_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	// Create test data
	userID := uuid.New()
	username := "Alice"
	lobbyID := uuid.New()
	joinCode := "ABC123"
	playerID := uuid.New()
	joinedAt := time.Now()

	// Expect query and return one row
	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).AddRow(lobbyID.String(), joinCode, models.LobbyStatusWaiting, userID.String(), playerID.String(), userID.String(), username, joinedAt, true)
	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(rows)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/internal/lobbies/"+lobbyID.String(), nil)

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyInternalHandler(repository.New(db))
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

func TestGetLobbyInternal_InvalidUUID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/internal/lobbies/invalid-uuid", nil)

	// Setup chi context for URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyInternalHandler(repository.New(db))
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLobbyInternal_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()

	// Expect query but return no rows
	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnRows(sqlmock.NewRows(columns))

	req := httptest.NewRequest(http.MethodGet, "/internal/lobbies/"+lobbyID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyInternalHandler(repository.New(db))
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobbyInternal_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()

	// Expect query but return database error
	mock.ExpectQuery("SELECT").WithArgs(lobbyID.String()).WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/internal/lobbies/"+lobbyID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("lobby_id", lobbyID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler := GetLobbyInternalHandler(repository.New(db))
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}
