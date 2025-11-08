package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/google/uuid"
)

func TestJoinLobby_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	lobbyID := uuid.New()
	joinCode := "ABC123"
	playerID := uuid.New()
	joinedAt := time.Now()

	// Transaction expectations
	mock.ExpectBegin()

	// Get lobby by join code
	lobby := models.Lobby{
		ID:        lobbyID,
		JoinCode:  joinCode,
		LeaderID:  uuid.New(),
		Status:    models.LobbyStatusWaiting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "join_code", "leader_id", "status", "created_at", "updated_at"}).
			AddRow(lobby.ID, lobby.JoinCode, lobby.LeaderID, lobby.Status, lobby.CreatedAt, lobby.UpdatedAt))

	// Get player count (currently 2 players in lobby)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Check if user is already a member (no)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(lobbyID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Create user if not exists
	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Add player to lobby
	mock.ExpectQuery("INSERT INTO players").
		WithArgs(lobbyID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(playerID.String(), joinedAt))

	mock.ExpectCommit()

	// Get lobby detail after commit
	mock.ExpectQuery("SELECT.*lobbies l.*").WithArgs(lobbyID).WillReturnRows(
		sqlmock.NewRows([]string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}).
			AddRow(lobbyID, joinCode, models.LobbyStatusWaiting, lobby.LeaderID, playerID.String(), userID.String(), username, joinedAt, true),
	)

	h := JoinLobbyHandler(repository.New(db))

	// Create request body
	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.LobbyDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate response structure
	if resp.LobbyID != lobbyID {
		t.Errorf("expected lobby_id %s, got %s", lobbyID, resp.LobbyID)
	}
	if resp.JoinCode != joinCode {
		t.Errorf("expected join_code %s, got %s", joinCode, resp.JoinCode)
	}
	if resp.Status != models.LobbyStatusWaiting {
		t.Errorf("expected status %s, got %s", models.LobbyStatusWaiting, resp.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestJoinLobby_MissingHeaders(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := JoinLobbyHandler(repository.New(db))

	// Missing both headers
	reqBody := models.JoinLobbyRequest{JoinCode: "ABC123"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Missing only X-User-ID
	req2 := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(headerUsername, "TestUser")

	rec2 := httptest.NewRecorder()
	h(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec2.Code)
	}
}

func TestJoinLobby_InvalidUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: "ABC123"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerUserID, "not-a-uuid")
	req.Header.Set(headerUsername, "TestUser")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestJoinLobby_InvalidRequestBody(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := JoinLobbyHandler(repository.New(db))

	userID := uuid.New()

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestJoinLobby_InvalidJoinCodeFormat(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := JoinLobbyHandler(repository.New(db))

	userID := uuid.New()

	// Join code too short
	reqBody := models.JoinLobbyRequest{JoinCode: "ABC"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Join code too long
	reqBody2 := models.JoinLobbyRequest{JoinCode: "ABC1234"}
	bodyBytes2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes2))

	rec2 := httptest.NewRecorder()
	h(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec2.Code)
	}
}

func TestJoinLobby_LobbyNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	joinCode := "INVALD"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestJoinLobby_LobbyNotJoinable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	lobbyID := uuid.New()
	joinCode := "RUNNNG"

	mock.ExpectBegin()

	// Get lobby by join code with status "running"
	lobby := models.Lobby{
		ID:        lobbyID,
		JoinCode:  joinCode,
		LeaderID:  uuid.New(),
		Status:    models.LobbyStatusInGame,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "join_code", "leader_id", "status", "created_at", "updated_at"}).
			AddRow(lobby.ID, lobby.JoinCode, lobby.LeaderID, lobby.Status, lobby.CreatedAt, lobby.UpdatedAt))

	mock.ExpectRollback()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestJoinLobby_LobbyFull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	lobbyID := uuid.New()
	joinCode := "FULLLO"

	mock.ExpectBegin()

	// Get lobby by join code
	lobby := models.Lobby{
		ID:        lobbyID,
		JoinCode:  joinCode,
		LeaderID:  uuid.New(),
		Status:    models.LobbyStatusWaiting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "join_code", "leader_id", "status", "created_at", "updated_at"}).
			AddRow(lobby.ID, lobby.JoinCode, lobby.LeaderID, lobby.Status, lobby.CreatedAt, lobby.UpdatedAt))

	// Get player count (6 players - full)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))

	mock.ExpectRollback()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestJoinLobby_UserAlreadyInLobby(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	lobbyID := uuid.New()
	joinCode := "INLOBB"

	mock.ExpectBegin()

	// Get lobby by join code
	lobby := models.Lobby{
		ID:        lobbyID,
		JoinCode:  joinCode,
		LeaderID:  uuid.New(),
		Status:    models.LobbyStatusWaiting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnRows(sqlmock.NewRows([]string{"id", "join_code", "leader_id", "status", "created_at", "updated_at"}).
			AddRow(lobby.ID, lobby.JoinCode, lobby.LeaderID, lobby.Status, lobby.CreatedAt, lobby.UpdatedAt))

	// Get player count (2 players)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Check if user is already a member (yes)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(lobbyID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectRollback()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestJoinLobby_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	joinCode := "ERRORR"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, join_code, leader_id, status, created_at, updated_at FROM lobbies WHERE join_code =").
		WithArgs(joinCode).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	h := JoinLobbyHandler(repository.New(db))

	reqBody := models.JoinLobbyRequest{JoinCode: joinCode}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/join", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}
