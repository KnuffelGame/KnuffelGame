package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
	"github.com/go-chi/chi/v5"
)

// SubscribeLobbyEvents returns an SSE handler for lobby subscriptions.
// Validates lobby_id (UUID v4), requires X-User-ID header (middleware to be added later),
// registers the connection in the registry, writes SSE frames, sends heartbeats,
// and performs cleanup on disconnect.
func SubscribeLobbyEvents(reg *models.Registry, cfg *models.Config, baseLog *slog.Logger) http.HandlerFunc {
	var (
		uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := requestLogger(ctx, baseLog).WithGroup("sse").With(
			slog.String("target_type", "lobby"),
		)

		// Validate lobby_id path param (UUID v4 format)
		lobbyID := chi.URLParam(r, "lobby_id")
		if lobbyID == "" || !uuidRe.MatchString(lobbyID) {
			BadRequest(w, "Invalid lobby_id format (must be UUID v4)", map[string]interface{}{"lobby_id": lobbyID}, log)
			return
		}
		log = log.With(slog.String("target_id", lobbyID))

		// Extract user id from headers (middleware to be integrated later)
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			Unauthorized(w, "Missing authentication (X-User-ID)", log)
			return
		}
		log = log.With(slog.String("user_id", userID))

		// Verify flusher
		flusher, ok := w.(http.Flusher)
		if !ok {
			InternalServerError(w, "Streaming unsupported (http.Flusher missing)", nil, log)
			return
		}

		// SSE headers (no retry directive per spec)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Initial flush to establish stream (no retry line per spec)
		flusher.Flush()

		// Register connection
		_, conn := reg.RegisterConnection(models.TargetType("lobby"), lobbyID, userID)
		log.Info("client connected")

		// Writer goroutine
		writeCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			for {
				select {
				case <-writeCtx.Done():
					return
				case <-conn.Done:
					return
				case msg, ok := <-conn.Ch:
					if !ok {
						return
					}
					writeSSE(w, msg.Event, msg.Data)
					flusher.Flush()
				}
			}
		}()

		// Heartbeat ticker
		hbInterval := time.Duration(30) * time.Second
		if cfg != nil && cfg.HeartbeatInterval > 0 {
			hbInterval = cfg.HeartbeatInterval
		}
		ticker := time.NewTicker(hbInterval)
		defer ticker.Stop()

		// Stream loop waits for client disconnect
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case <-conn.Done:
				break loop
			case t := <-ticker.C:
				// Numeric epoch milliseconds for timestamp
				payload := map[string]interface{}{"timestamp": t.UTC().UnixNano() / 1000000}
				data, _ := json.Marshal(payload)
				// non-blocking heartbeat send to avoid stalls
				select {
				case conn.Ch <- models.SSEMessage{Event: "keep_alive", Data: data}:
				default:
					// drop heartbeat if backpressure; cleanup will be handled by writer or broadcast failures
				}
			}
		}

		// Cleanup
		reg.RemoveConnection(models.TargetType("lobby"), lobbyID, userID)
		log.Info("client disconnected")
	}
}

// writeSSE writes a single SSE event frame.
func writeSSE(w http.ResponseWriter, event string, data []byte) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	if len(data) > 0 {
		fmt.Fprintf(w, "data: %s\n\n", string(data))
	} else {
		fmt.Fprint(w, "data: {}\n\n")
	}
}

// requestLogger obtains a request-scoped logger from context or falls back to base.
func requestLogger(ctx context.Context, base *slog.Logger) *slog.Logger {
	l := logger.Logger(ctx)
	if l == nil {
		l = base
	}
	if l == nil {
		l = logger.Default()
	}
	return l
}
