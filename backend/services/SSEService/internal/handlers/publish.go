package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// PublishEventRequest mirrors the OpenAPI Variant B schema for event publishing.
type PublishEventRequest struct {
	LobbyID   string                 `json:"lobby_id"`       // UUID v4 lobby identifier
	EventType string                 `json:"event_type"`     // event name (1-128 chars, not "keep_alive")
	Data      map[string]interface{} `json:"data,omitempty"` // optional payload object
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

		// Validate required fields
		if req.LobbyID == "" {
			BadRequest(w, "Missing required field: lobby_id", nil, log)
			return
		}

		eventType := req.EventType
		if eventType == "" {
			BadRequest(w, "Missing required field: event_type", nil, log)
			return
		}

		// Validate event_type constraints
		eventType = eventType
		if len(eventType) > 128 {
			BadRequest(w, "event_type must be ≤128 characters", map[string]interface{}{"length": len(eventType)}, log)
			return
		}

		if eventType == "keep_alive" {
			BadRequest(w, "event_type 'keep_alive' is reserved and cannot be published", map[string]interface{}{"event_type": eventType}, log)
			return
		}

		// Validate data constraints
		var finalData map[string]interface{}
		if req.Data != nil {
			// Data is already validated by JSON decoder to be a map[string]interface{}
			finalData = req.Data
		} else {
			// If data is missing, create {"timestamp": epoch_ms}
			finalData = make(map[string]interface{})
		}

		// Inject/overwrite timestamp with current epoch milliseconds
		epochMs := time.Now().UTC().UnixNano() / 1000000
		finalData["timestamp"] = epochMs

		// Marshal the final data
		dataBytes, err := json.Marshal(finalData)
		if err != nil {
			InternalServerError(w, "Failed to marshal payload", map[string]interface{}{"detail": err.Error()}, log)
			return
		}

		// Validate lobby exists in registry
		if !reg.HasTarget(models.TargetTypeLobby, req.LobbyID) {
			NotFound(w, "Target lobby not found", log)
			return
		}

		// Broadcast to all connections in lobby
		found, sent, failed, ok := reg.Broadcast(models.TargetTypeLobby, req.LobbyID, models.SSEMessage{
			Event: eventType,
			Data:  dataBytes,
		}, "")

		if !ok {
			// This should not happen since we checked HasTarget above, but be safe
			NotFound(w, "Target lobby not found", log)
			return
		}

		log.Info("publish completed",
			slog.String("lobby_id", req.LobbyID),
			slog.String("event_type", eventType),
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
