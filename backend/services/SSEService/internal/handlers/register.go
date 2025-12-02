package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// RegisterTargetRequest mirrors OpenAPI schema: target_type + target_id.
type RegisterTargetRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

// SuccessResponse is a simple success envelope.
type SuccessResponse struct {
	Success    bool   `json:"success"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
}

// RegisterTarget creates a target entry in the registry if absent; 409 if already exists.
func RegisterTarget(reg *models.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := requestLogger(r.Context(), nil).WithGroup("internal").With(slog.String("action", "register"))

		var req RegisterTargetRequest
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

		// 409 if already present
		if reg.HasTarget(models.TargetType(tt), id) {
			httpx.WriteError(w, http.StatusConflict, "already_exists", "Target already registered", map[string]interface{}{"target_type": tt, "target_id": id}, log)
			return
		}

		reg.EnsureTarget(models.TargetType(tt), id)
		httpx.WriteJSON(w, http.StatusOK, SuccessResponse{Success: true, TargetType: tt, TargetID: id}, log)
	}
}
