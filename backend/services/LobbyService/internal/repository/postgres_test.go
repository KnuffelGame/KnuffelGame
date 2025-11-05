package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/models"
	"github.com/google/uuid"
)

func TestCreateUserIfNotExistsTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := New(db)
	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	userID := uuid.New()
	username := "bob"

	mock.ExpectExec("INSERT INTO users").WithArgs(userID, username).WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.CreateUserIfNotExistsTx(tx, userID, username); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}

	// rollback to clean up
	tx.Rollback()
}

func TestCreateLobbyTxAndAddPlayerTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := New(db)
	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leaderID := uuid.New()
	joinCode := "ABC123"
	lobbyID := uuid.New()
	playerID := uuid.New()
	joinedAt := time.Now()

	// Expect lobby insert returning id
	mock.ExpectQuery("INSERT INTO lobbies").WithArgs(joinCode, leaderID, models.LobbyStatusWaiting).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(lobbyID.String()))

	// Expect players insert returning id and joined_at
	mock.ExpectQuery("INSERT INTO players").WithArgs(lobbyID, leaderID).WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(playerID.String(), joinedAt))

	lid, err := repo.CreateLobbyTx(tx, joinCode, leaderID)
	if err != nil {
		t.Fatalf("CreateLobbyTx error: %v", err)
	}
	if lid != lobbyID {
		t.Fatalf("expected lobby id %v, got %v", lobbyID, lid)
	}

	pid, ja, err := repo.AddPlayerTx(tx, lobbyID, leaderID)
	if err != nil {
		t.Fatalf("AddPlayerTx error: %v", err)
	}
	if pid != playerID {
		t.Fatalf("expected player id %v, got %v", playerID, pid)
	}
	if !ja.Equal(joinedAt) && ja.Sub(joinedAt) > time.Second {
		t.Fatalf("expected joinedAt close to %v, got %v", joinedAt, ja)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}

	tx.Rollback()
}

func TestGetLobbyLeaderIDAndIsMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	lobbyID := uuid.New()
	leaderID := uuid.New()

	mock.ExpectQuery("SELECT leader_id::text FROM lobbies").WithArgs(lobbyID).WillReturnRows(sqlmock.NewRows([]string{"leader_id"}).AddRow(leaderID.String()))

	gotLeader, err := repo.GetLobbyLeaderID(ctx, lobbyID)
	if err != nil {
		t.Fatalf("GetLobbyLeaderID error: %v", err)
	}
	if gotLeader != leaderID {
		t.Fatalf("expected leader %v, got %v", leaderID, gotLeader)
	}

	// IsMember
	userID := uuid.New()
	mock.ExpectQuery("SELECT EXISTS").WithArgs(lobbyID, userID).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.IsMember(ctx, lobbyID, userID)
	if err != nil {
		t.Fatalf("IsMember error: %v", err)
	}
	if !exists {
		t.Fatalf("expected IsMember true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestGetLobbyDetail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := New(db)
	ctx := context.Background()

	lobbyID := uuid.New()
	joinCode := "XYZ789"
	status := models.LobbyStatusWaiting
	leaderID := uuid.New()
	playerID := uuid.New()
	playerUserID := uuid.New()
	username := "Alice"
	joinedAt := time.Now()

	columns := []string{"lobby_id", "join_code", "status", "leader_id", "player_id", "user_id", "username", "joined_at", "is_active"}
	rows := sqlmock.NewRows(columns).
		AddRow(lobbyID.String(), joinCode, status, leaderID.String(), playerID.String(), playerUserID.String(), username, joinedAt, true)

	mock.ExpectQuery("SELECT").WithArgs(lobbyID).WillReturnRows(rows)

	resp, err := repo.GetLobbyDetail(ctx, lobbyID)
	if err != nil {
		t.Fatalf("GetLobbyDetail error: %v", err)
	}
	if resp.LobbyID != lobbyID {
		t.Fatalf("expected lobby id %v, got %v", lobbyID, resp.LobbyID)
	}
	if resp.JoinCode != joinCode {
		t.Fatalf("expected joinCode %s, got %s", joinCode, resp.JoinCode)
	}
	if resp.Status != status {
		t.Fatalf("expected status %s, got %s", status, resp.Status)
	}
	if resp.LeaderID != leaderID {
		t.Fatalf("expected leader %v, got %v", leaderID, resp.LeaderID)
	}
	if len(resp.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(resp.Players))
	}
	p := resp.Players[0]
	if p.ID != playerID {
		t.Fatalf("expected player id %v, got %v", playerID, p.ID)
	}
	if p.UserID != playerUserID {
		t.Fatalf("expected player user id %v, got %v", playerUserID, p.UserID)
	}
	if p.Username != username {
		t.Fatalf("expected username %s, got %s", username, p.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}
