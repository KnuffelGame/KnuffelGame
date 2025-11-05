package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestAuthMiddleware_Success(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(DefaultHeaderUserID, userID.String())
	req.Header.Set(DefaultHeaderUsername, "alice")

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		u, ok := FromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if u.ID != userID {
			t.Fatalf("expected user id %s, got %s", userID, u.ID)
		}
		if u.Username != "alice" {
			t.Fatalf("expected username alice, got %s", u.Username)
		}
		w.WriteHeader(http.StatusOK)
	})

	h := AuthMiddleware(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// no headers set

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called when headers are missing")
	})

	h := AuthMiddleware(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidUUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(DefaultHeaderUserID, "not-a-uuid")
	req.Header.Set(DefaultHeaderUsername, "bob")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called when UUID is invalid")
	})

	h := AuthMiddleware(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestNewAuthMiddleware_CustomHeaders(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-My-ID", userID.String())
	req.Header.Set("X-My-Name", "carol")

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		u, ok := FromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if u.ID != userID {
			t.Fatalf("expected user id %s, got %s", userID, u.ID)
		}
		if u.Username != "carol" {
			t.Fatalf("expected username carol, got %s", u.Username)
		}
	})

	h := NewAuthMiddleware("X-My-ID", "X-My-Name")(next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusOK && rec.Code != 0 {
		// Some handlers don't explicitly write status; ensure not an error
		t.Fatalf("unexpected status code %d", rec.Code)
	}
}
