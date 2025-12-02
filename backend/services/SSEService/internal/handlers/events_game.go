package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
	"github.com/go-chi/chi/v5"
)

// SubscribeGameEvents returns an SSE handler for game subscriptions.
// Mirrors lobby behavior with game-specific ID validation.
func SubscribeGameEvents(reg *models.Registry, cfg *models.Config, baseLog *slog.Logger) http.HandlerFunc {
	var (
		idRe = regexp.MustCompile(`^gam_[a-zA-Z0-9]+$`)
	)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := requestLogger(ctx, baseLog).WithGroup("sse").With(
			slog.String("target_type", "game"),
		)

		// Validate game_id path param
		gameID := chi.URLParam(r, "game_id")
		if gameID == "" || !idRe.MatchString(gameID) {
			BadRequest(w, "Invalid game_id format", map[string]interface{}{"game_id": gameID}, log)
			return
		}
		log = log.With(slog.String("target_id", gameID))

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

		// SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Disable proxy buffering (nginx)
		w.Header().Set("X-Accel-Buffering", "no")

		// Initial flush to establish stream
		writeRetryLine(w, 5000)
		flusher.Flush()

		// Register connection
		_, conn := reg.RegisterConnection(models.TargetType("game"), gameID, userID)
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
				payload := map[string]string{"timestamp": t.UTC().Format(time.RFC3339)}
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
		reg.RemoveConnection(models.TargetType("game"), gameID, userID)
		log.Info("client disconnected")
	}
}
