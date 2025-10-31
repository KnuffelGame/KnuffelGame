package logger

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ChiMiddleware returns a chi compatible middleware that logs each HTTP request on completion.
// It expects a *slog.Logger; you can pass logger.Default() if desired.
// The logger is injected into the request context for handlers via WithLogger.
// Attributes under group "http": method, path, status, duration_ms, request_id, remote_ip, user_agent.
func ChiMiddleware(l *slog.Logger) func(next http.Handler) http.Handler {
	if l == nil {
		l = Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithLogger(r.Context(), l)
			r = r.WithContext(ctx)
			start := time.Now()
			rid := requestID(r)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			durMs := float64(time.Since(start)) / float64(time.Millisecond)
			Logger(ctx).WithGroup("http").Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Float64("duration_ms", durMs),
				slog.String("request_id", rid),
				slog.String("remote_ip", remoteIP(r)),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// requestID returns header X-Request-ID or generates a random 16-byte hex.
func requestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-ID"); v != "" {
		return v
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// remoteIP attempts to extract the client IP considering standard proxy headers.
func remoteIP(r *http.Request) string {
	// check X-Forwarded-For first; take first component
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// may contain multiple comma separated IPs
		for i := 0; i < len(xff); i++ { // manual parse to avoid strings.Split allocs
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	// fallback remote addr (host:port)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
