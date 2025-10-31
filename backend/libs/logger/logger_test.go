package logger

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewLoggerColor(t *testing.T) {
	var buf bytes.Buffer
	l := New(WithWriter(&buf), WithColor(true), WithLevel(slog.LevelDebug), WithService("TestSvc"))
	l.Info("hello", slog.String("k", "v"))
	out := buf.String()
	if !strings.Contains(out, "\"msg\":\"hello\"") {
		t.Fatalf("expected msg attribute, got %s", out)
	}
	if !strings.Contains(out, "TestSvc") {
		t.Fatalf("expected service attribute")
	}
	if !strings.Contains(out, "\x1b[32m") { // green color for info
		t.Fatalf("expected ANSI color codes for info level, got %q", out)
	}
}

func TestChiMiddleware(t *testing.T) {
	var buf bytes.Buffer
	l := New(WithWriter(&buf), WithColor(false))
	mw := ChiMiddleware(l)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(201)
	}))
	r := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Fatalf("handler not called")
	}
	if w.Result().StatusCode != 201 {
		t.Fatalf("unexpected status")
	}
	out := buf.String()
	if !strings.Contains(out, "\"status\":201") {
		t.Fatalf("expected status attr in log: %s", out)
	}
	if !strings.Contains(out, "\"path\":\"/x\"") {
		t.Fatalf("expected path attr")
	}
}
