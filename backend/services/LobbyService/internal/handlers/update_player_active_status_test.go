package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestUpdatePlayerActiveStatus_Success_ActiveToInactive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()
	playerID := uuid.New()

	// Transaction expectations
	mock.ExpectBegin()

	// Get lobby leader ID (to verify lobby exists)
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(uuid.New().String()))

	// Check if player is member
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM players WHERE lobby_id = \\$1 AND user_id = \\$2\\)").
		WithArgs(lobbyID, playerID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Update player active status
	mock.ExpectExec("UPDATE players SET is_active = \\$1 WHERE lobby_id = \\$2 AND id = \\$3").
		WithArgs(false, lobbyID, playerID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: false}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+lobbyID.String()+"/players/"+playerID.String()+"/active-status", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id", "player_id"},
			Values: []string{lobbyID.String(), playerID.String()},
		},
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestUpdatePlayerActiveStatus_Success_InactiveToActive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()
	playerID := uuid.New()

	// Transaction expectations
	mock.ExpectBegin()

	// Get lobby leader ID (to verify lobby exists)
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(uuid.New().String()))

	// Check if player is member
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM players WHERE lobby_id = \\$1 AND user_id = \\$2\\)").
		WithArgs(lobbyID, playerID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Update player active status
	mock.ExpectExec("UPDATE players SET is_active = \\$1 WHERE lobby_id = \\$2 AND id = \\$3").
		WithArgs(true, lobbyID, playerID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+lobbyID.String()+"/players/"+playerID.String()+"/active-status", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id", "player_id"},
			Values: []string{lobbyID.String(), playerID.String()},
		},
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestUpdatePlayerActiveStatus_InvalidLobbyID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/not-a-uuid/players/"+uuid.New().String()+"/active-status", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdatePlayerActiveStatus_InvalidPlayerID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+uuid.New().String()+"/players/not-a-uuid/active-status", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdatePlayerActiveStatus_LobbyNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()
	playerID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+lobbyID.String()+"/players/"+playerID.String()+"/active-status", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id", "player_id"},
			Values: []string{lobbyID.String(), playerID.String()},
		},
	}))
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

func TestUpdatePlayerActiveStatus_PlayerNotFoundInLobby(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()
	playerID := uuid.New()

	mock.ExpectBegin()

	// Get lobby leader ID (lobby exists)
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(uuid.New().String()))

	// Check if player is member (not a member)
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM players WHERE lobby_id = \\$1 AND user_id = \\$2\\)").
		WithArgs(lobbyID, playerID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectRollback()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+lobbyID.String()+"/players/"+playerID.String()+"/active-status", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id", "player_id"},
			Values: []string{lobbyID.String(), playerID.String()},
		},
	}))
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

func TestUpdatePlayerActiveStatus_InvalidRequestBody(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+uuid.New().String()+"/players/"+uuid.New().String()+"/active-status", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdatePlayerActiveStatus_DatabaseTransactionError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	lobbyID := uuid.New()
	playerID := uuid.New()

	mock.ExpectBegin()

	// Get lobby leader ID (lobby exists)
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(uuid.New().String()))

	// Check if player is member
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM players WHERE lobby_id = \\$1 AND user_id = \\$2\\)").
		WithArgs(lobbyID, playerID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Update player active status fails
	mock.ExpectExec("UPDATE players SET is_active = \\$1 WHERE lobby_id = \\$2 AND id = \\$3").
		WithArgs(true, lobbyID, playerID).
		WillReturnError(sql.ErrConnDone)

	mock.ExpectRollback()

	h := UpdatePlayerActiveStatusHandler(repository.New(db))

	reqBody := models.UpdatePlayerActiveStatusRequest{IsActive: true}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/lobbies/"+lobbyID.String()+"/players/"+playerID.String()+"/active-status", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id", "player_id"},
			Values: []string{lobbyID.String(), playerID.String()},
		},
	}))
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
