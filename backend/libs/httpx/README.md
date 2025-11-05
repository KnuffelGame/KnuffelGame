# httpx Library

A lightweight HTTP response utility library for Go web services. Provides standardized JSON responses and error handling for HTTP handlers.

## Features

- 📦 **Standard JSON Responses**: Consistent JSON encoding with proper headers
- ❌ **Error Handling**: Standardized error payload structure across all services
- 🎯 **Convenience Methods**: Common HTTP status codes with dedicated functions
- 🧪 **Well Tested**: Comprehensive test coverage
- 🔌 **Zero Dependencies**: Only uses Go standard library

## Installation

This is an internal library. Add it to your service's `go.mod` with a replace directive:

```go
require github.com/KnuffelGame/KnuffelGame/backend/libs/httpx v0.0.0

replace github.com/KnuffelGame/KnuffelGame/backend/libs/httpx => ../../libs/httpx
```

## Usage

### Basic JSON Response

```go
import "github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"

func MyHandler(w http.ResponseWriter, r *http.Request) {
    log := logger.FromContext(r.Context())
    
    data := map[string]string{"message": "hello world"}
    httpx.WriteJSON(w, http.StatusOK, data, log)
}
```

### Error Responses

Standard error response format:

```go
httpx.WriteError(w, http.StatusBadRequest, "invalid_input", "Username is required", nil, log)
```

Response:
```json
{
  "error": "invalid_input",
  "message": "Username is required"
}
```

With details:

```go
details := map[string]interface{}{
    "field": "email",
    "provided": "invalid-email"
}
httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "Invalid email format", details, log)
```

Response:
```json
{
  "error": "validation_failed",
  "message": "Invalid email format",
  "details": {
    "field": "email",
    "provided": "invalid-email"
  }
}
```

### Convenience Methods

Common HTTP status codes have dedicated helper functions:

```go
// 400 Bad Request
httpx.WriteBadRequest(w, "Invalid request body", details, log)

// 401 Unauthorized
httpx.WriteUnauthorized(w, "Authentication required", log)

// 403 Forbidden
httpx.WriteForbidden(w, "Access denied", log)

// 404 Not Found
httpx.WriteNotFound(w, "User not found", log)

// 500 Internal Server Error
httpx.WriteInternalError(w, "Database connection failed", details, log)

// 204 No Content
httpx.WriteNoContent(w)
```

### Decode JSON Request Body

```go
type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
    log := logger.FromContext(r.Context())
    
    var req CreateUserRequest
    if err := httpx.DecodeJSON(r, &req); err != nil {
        httpx.WriteBadRequest(w, "Invalid JSON body", 
            map[string]interface{}{"detail": err.Error()}, log)
        return
    }
    
    // Process request...
}
```

## API Reference

### Response Functions

- `WriteJSON(w, status, payload, log)` - Write any JSON response
- `WriteError(w, status, code, message, details, log)` - Write error response
- `WriteBadRequest(w, message, details, log)` - 400 Bad Request
- `WriteUnauthorized(w, message, log)` - 401 Unauthorized
- `WriteForbidden(w, message, log)` - 403 Forbidden
- `WriteNotFound(w, message, log)` - 404 Not Found
- `WriteInternalError(w, message, details, log)` - 500 Internal Server Error
- `WriteNoContent(w)` - 204 No Content

### Request Functions

- `DecodeJSON(r, target)` - Decode request body as JSON

### Types

#### ErrorPayload

```go
type ErrorPayload struct {
    Error   string                 `json:"error"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

- `Error`: Machine-readable error code (e.g., "invalid_input", "not_found")
- `Message`: Human-readable error message
- `Details`: Optional additional context

## Best Practices

1. **Always pass a logger**: Helps debug JSON encoding failures
2. **Use consistent error codes**: Define error code constants in your service
3. **Provide meaningful messages**: Help clients understand what went wrong
4. **Use details sparingly**: Only include relevant debugging information
5. **Use convenience methods**: They ensure consistency across your service

## Testing

Run tests:

```bash
cd backend/libs/httpx
go test -v
```

## Examples

### Complete Handler Example

```go
func HandleCreateToken(w http.ResponseWriter, r *http.Request) {
    log := logger.FromContext(r.Context())
    
    var req CreateTokenRequest
    if err := httpx.DecodeJSON(r, &req); err != nil {
        httpx.WriteBadRequest(w, "Invalid JSON body", 
            map[string]interface{}{"detail": err.Error()}, log)
        return
    }
    
    if err := req.Validate(); err != nil {
        httpx.WriteBadRequest(w, "Validation failed", 
            map[string]interface{}{"fields": err}, log)
        return
    }
    
    token, err := generateToken(req.Username, req.IsGuest)
    if err != nil {
        httpx.WriteInternalError(w, "Failed to generate token", 
            map[string]interface{}{"detail": err.Error()}, log)
        return
    }
    
    httpx.WriteJSON(w, http.StatusOK, 
        CreateTokenResponse{Token: token}, log)
}
```

## Version

v0.0.0 - November 2025

