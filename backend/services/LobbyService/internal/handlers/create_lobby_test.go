package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/joincode"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/repository"
	"github.com/google/uuid"
)

func TestCreateLobby_Success(t *testing.T) {
	// create mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"

	// Expectations: transaction begin, insert user (exec), joincode existence check, insert lobby returning id, insert player returning id and joined_at, commit
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	// joincode generator checks for existence (use permissive regex)
	mock.ExpectQuery("SELECT EXISTS").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	lobbyID := uuid.New()
	mock.ExpectQuery("INSERT INTO lobbies").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(lobbyID.String()))

	playerID := uuid.New()
	joinedAt := time.Now()
	mock.ExpectQuery("INSERT INTO players").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(
		sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(playerID.String(), joinedAt),
	)
	mock.ExpectCommit()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(repository.New(db), codeGen)

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

	// ensure expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestCreateLobby_MissingUserID(t *testing.T) {
	// When headers are missing, handler returns early and DB isn't used. Create a mock DB but don't set expectations.
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(repository.New(db), codeGen)

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
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(repository.New(db), codeGen)

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
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(repository.New(db), codeGen)

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
	// Test creating two lobbies with same user; set expectations for two transactions
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock DB: %v", err)
	}
	defer db.Close()

	userID := uuid.New()
	username := "TestUser"

	// First lobby expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT EXISTS").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	lobby1 := uuid.New()
	mock.ExpectQuery("INSERT INTO lobbies").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(lobby1.String()))
	player1 := uuid.New()
	mock.ExpectQuery("INSERT INTO players").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(player1.String(), time.Now()))
	mock.ExpectCommit()

	// Second lobby expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT EXISTS").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	lobby2 := uuid.New()
	mock.ExpectQuery("INSERT INTO lobbies").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(lobby2.String()))
	player2 := uuid.New()
	mock.ExpectQuery("INSERT INTO players").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(player2.String(), time.Now()))
	mock.ExpectCommit()

	codeGen := joincode.NewGenerator(db)
	h := CreateLobbyHandler(repository.New(db), codeGen)

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

	// Create second lobby with same user
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
	if resp1.LeaderID != resp2.LeaderID {
		t.Error("expected same leader ID")
	}

	// ensure expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}
