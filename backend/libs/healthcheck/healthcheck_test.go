package healthcheck_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
	"github.com/go-chi/chi/v5"
)

func TestMount(t *testing.T) {
	r := chi.NewRouter()
	healthcheck.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected Content-Type text/plain; charset=utf-8, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "1" {
		t.Fatalf("expected body '1', got %q", string(body))
	}
}

func TestHandler(t *testing.T) {
	h := healthcheck.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected Content-Type text/plain; charset=utf-8, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "1" {
		t.Fatalf("expected body '1', got %q", string(body))
	}
}
