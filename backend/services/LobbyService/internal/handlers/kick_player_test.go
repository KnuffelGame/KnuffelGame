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

func TestKickPlayer_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "LeaderUser"
	lobbyID := uuid.New()
	targetUserID := uuid.New()

	// Transaction expectations
	mock.ExpectBegin()

	// Get lobby leader ID
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(userID.String()))

	// Check if target user is member
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(lobbyID, targetUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// Kick player
	mock.ExpectExec("UPDATE players SET is_active = false, left_at = NOW\\(\\) WHERE lobby_id = \\$1 AND user_id = \\$2 AND is_active = true").
		WithArgs(lobbyID, targetUserID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: targetUserID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
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

func TestKickPlayer_MissingHeaders(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := KickPlayerHandler(repository.New(db))

	// Missing both headers
	reqBody := models.KickPlayerRequest{TargetUserID: uuid.New().String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+uuid.New().String()+"/kick", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Missing only X-User-ID
	req2 := httptest.NewRequest(http.MethodPost, "/lobbies/"+uuid.New().String()+"/kick", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(headerUsername, "TestUser")

	rec2 := httptest.NewRecorder()
	h(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec2.Code)
	}
}

func TestKickPlayer_InvalidUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: uuid.New().String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+uuid.New().String()+"/kick", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, "not-a-uuid")
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestKickPlayer_InvalidLobbyID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := KickPlayerHandler(repository.New(db))

	userID := uuid.New()
	reqBody := models.KickPlayerRequest{TargetUserID: uuid.New().String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/not-a-uuid/kick", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestKickPlayer_InvalidRequestBody(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := KickPlayerHandler(repository.New(db))

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+uuid.New().String()+"/kick", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestKickPlayer_InvalidTargetUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	h := KickPlayerHandler(repository.New(db))

	userID := uuid.New()
	reqBody := models.KickPlayerRequest{TargetUserID: "not-a-uuid"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+uuid.New().String()+"/kick", bytes.NewReader(bodyBytes))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, "TestUser")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestKickPlayer_LobbyNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"
	lobbyID := uuid.New()
	targetUserID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: targetUserID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
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

func TestKickPlayer_UserNotLeader(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "NotLeader"
	lobbyID := uuid.New()
	targetUserID := uuid.New()
	leaderID := uuid.New() // Different from userID

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(leaderID.String()))
	mock.ExpectRollback()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: targetUserID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestKickPlayer_TargetNotInLobby(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "LeaderUser"
	lobbyID := uuid.New()
	targetUserID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(userID.String()))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(lobbyID, targetUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: targetUserID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
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

func TestKickPlayer_CannotKickSelf(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "LeaderUser"
	lobbyID := uuid.New()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: userID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerUsername, username)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestKickPlayer_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "LeaderUser"
	lobbyID := uuid.New()
	targetUserID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT leader_id::text FROM lobbies WHERE id =").
		WithArgs(lobbyID).
		WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(userID.String()))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(lobbyID, targetUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE players SET is_active = false, left_at = NOW\\(\\) WHERE lobby_id = \\$1 AND user_id = \\$2 AND is_active = true").
		WithArgs(lobbyID, targetUserID).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	h := KickPlayerHandler(repository.New(db))

	reqBody := models.KickPlayerRequest{TargetUserID: targetUserID.String()}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/lobbies/"+lobbyID.String()+"/kick", bytes.NewReader(bodyBytes))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"lobby_id"},
			Values: []string{lobbyID.String()},
		},
	}))
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
