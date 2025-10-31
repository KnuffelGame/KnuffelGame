package models

// CreateTokenRequest represents the incoming JSON payload for /internal/create
// If user_id omitted, service will generate a UUID.
// {"username":"Alice"} or {"user_id":"<uuid>","username":"Alice"}

type CreateTokenRequest struct {
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username"`
}

// CreateTokenResponse is the success response containing the signed JWT token.

type CreateTokenResponse struct {
	Token string `json:"token"`
}

// ErrorResponse matches the shared error schema.

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}
