package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// UnregisterTargetRequest represents the payload to close and remove a target.
type UnregisterTargetRequest struct {
	TargetType string `json:"target_type"` // "lobby" | "game"
	TargetID   string `json:"target_id"`   // lby_* or gam_*
	Reason     string `json:"reason"`      // enum (free-form for now): e.g., "service_shutdown","target_removed","normal"
}

// UnregisterTargetResponse contains the result of the unregister operation.
type UnregisterTargetResponse struct {
	Success           bool   `json:"success"`
	ConnectionsClosed int    `json:"connections_closed"`
	Message           string `json:"message"`
	Timestamp         string `json:"timestamp"`
}

// UnregisterTarget handles POST /internal/unregister:
// - validates payload
// - emits a final "service_close" event with {"reason": ...}
// - closes all connections and removes the target
func UnregisterTarget(reg *models.Registry, baseLog *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := requestLogger(r.Context(), baseLog).WithGroup("internal").With(slog.String("action", "unregister"))

		var req UnregisterTargetRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			BadRequest(w, "Invalid JSON body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		tt := strings.ToLower(strings.TrimSpace(req.TargetType))
		if tt != "lobby" && tt != "game" {
			BadRequest(w, "Invalid target_type", map[string]interface{}{"allowed": []string{"lobby", "game"}}, log)
			return
		}
		id := strings.TrimSpace(req.TargetID)
		if id == "" {
			BadRequest(w, "target_id is required", nil, log)
			return
		}
		reason := strings.TrimSpace(req.Reason)
		if reason == "" {
			BadRequest(w, "reason is required", nil, log)
			return
		}

		// Emit final event to connected clients before closing
		payload := map[string]string{"reason": reason}
		data, err := json.Marshal(payload)
		if err != nil {
			InternalServerError(w, "Failed to marshal close payload", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		targetType := models.TargetType(tt)

		// Best-effort broadcast; if target not found, respond 404 according to spec
		_, _, _, ok := reg.Broadcast(targetType, id, models.SSEMessage{
			Event: "service_close",
			Data:  data,
		}, "")

		if !ok {
			httpx.WriteNotFound(w, "Target not found", log)
			return
		}

		// Close and remove
		closed, _ := reg.UnregisterTarget(targetType, id)

		log.Info("unregistered target",
			slog.String("target_type", tt),
			slog.String("target_id", id),
			slog.Int("connections_closed", closed),
			slog.String("reason", reason),
		)

		resp := UnregisterTargetResponse{
			Success:           true,
			ConnectionsClosed: closed,
			Message:           "target unregistered and connections closed",
			Timestamp:         time.Now().UTC().Format(time.RFC3339),
		}
		httpx.WriteJSON(w, http.StatusOK, resp, log)
	}
}
