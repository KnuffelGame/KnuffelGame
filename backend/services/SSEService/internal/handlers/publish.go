package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// PublishEventRequest mirrors the OpenAPI schema for event publishing.
type PublishEventRequest struct {
	TargetType   string                 `json:"target_type"`              // "lobby" | "game"
	TargetID     string                 `json:"target_id"`                // lby_* or gam_*
	EventType    string                 `json:"event_type"`               // event name
	TargetUserID string                 `json:"target_user_id,omitempty"` // optional, direct message
	Data         map[string]interface{} `json:"data"`                     // payload
}

// PublishEventResponse is returned on successful publish.
type PublishEventResponse struct {
	Success           bool `json:"success"`
	ConnectionsFound  int  `json:"connections_found"`
	EventsSent        int  `json:"events_sent"`
	FailedConnections int  `json:"failed_connections"`
}

// Publish handles POST /internal/publish using the registry for broadcast.
func Publish(reg *models.Registry, baseLog *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := requestLogger(r.Context(), baseLog).WithGroup("internal").With(slog.String("action", "publish"))

		var req PublishEventRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			BadRequest(w, "Invalid JSON body", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		tt := strings.ToLower(strings.TrimSpace(req.TargetType))
		if tt != "lobby" && tt != "game" {
			BadRequest(w, "Invalid target_type", map[string]interface{}{"allowed": []string{"lobby", "game"}}, log)
			return
		}
		if strings.TrimSpace(req.TargetID) == "" {
			BadRequest(w, "target_id is required", nil, log)
			return
		}
		if strings.TrimSpace(req.EventType) == "" {
			BadRequest(w, "event_type is required", nil, log)
			return
		}
		if req.Data == nil || len(req.Data) == 0 {
			BadRequest(w, "data is required", nil, log)
			return
		}

		dataBytes, err := json.Marshal(req.Data)
		if err != nil {
			InternalServerError(w, "Failed to marshal payload", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		targetType := models.TargetType(tt)
		found, sent, failed, ok := reg.Broadcast(targetType, req.TargetID, models.SSEMessage{
			Event: req.EventType,
			Data:  dataBytes,
		}, strings.TrimSpace(req.TargetUserID))

		if !ok {
			// Specific error code per spec
			httpx.WriteError(w, http.StatusNotFound, "target_not_found", "Target entry not found", map[string]interface{}{"target_type": tt, "target_id": req.TargetID}, log)
			return
		}

		log.Info("publish completed",
			slog.String("target_type", tt),
			slog.String("target_id", req.TargetID),
			slog.String("event_type", req.EventType),
			slog.Int("connections_found", found),
			slog.Int("events_sent", sent),
			slog.Int("failed_connections", failed),
		)

		resp := PublishEventResponse{
			Success:           true,
			ConnectionsFound:  found,
			EventsSent:        sent,
			FailedConnections: failed,
		}
		httpx.WriteJSON(w, http.StatusOK, resp, log)
	}
}
