# Backend Libraries

This directory contains shared Go libraries used across all KnuffelGame backend services. These libraries provide common functionality, reduce code duplication, and ensure consistency across the microservices architecture.

## Overview

The `libs` directory contains four core libraries that form the foundation of our backend services:

- **[auth](./auth/)** - Authentication middleware for user identification
- **[healthcheck](./healthcheck/)** - Health check endpoint utilities  
- **[httpx](./httpx/)** - HTTP response and error handling utilities
- **[logger](./logger/)** - Structured logging with request middleware

## Architecture Philosophy

These libraries follow a few key principles:

- **Minimal Dependencies**: Each library uses only Go standard library dependencies where possible
- **Lightweight**: No heavy frameworks or external dependencies
- **Reusable**: Designed to be used across all backend services
- **Consistent**: Provide standardized patterns for common operations
- **Context-Aware**: Work seamlessly with Go's `context.Context`

## Quick Start

Each service should include these libraries using Go modules with replace directives:

```go
require (
    github.com/KnuffelGame/KnuffelGame/backend/libs/auth v0.0.0
    github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck v0.0.0
    github.com/KnuffelGame/KnuffelGame/backend/libs/httpx v0.0.0
    github.com/KnuffelGame/KnuffelGame/backend/libs/logger v0.0.0
)

replace github.com/KnuffelGame/KnuffelGame/backend/libs/auth => ../../libs/auth
replace github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck => ../../libs/healthcheck
replace github.com/KnuffelGame/KnuffelGame/backend/libs/httpx => ../../libs/httpx
replace github.com/KnuffelGame/KnuffelGame/backend/libs/logger => ../../libs/logger
```

## Library Dependencies

The libraries have the following dependency relationships:

```
logger (base library)
    ↓
    ↓ (optional)
auth -----
    ↓         ↓
httpx <-----
    ↓
healthcheck (standalone)
```

- **logger** is the foundational library - all other libraries can optionally use it
- **auth** uses logger for warnings
- **httpx** can optionally use logger for error logging
- **healthcheck** is completely standalone

## Service Integration Pattern

Most services follow this pattern when integrating these libraries:

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/auth"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck"
)

func main() {
    // 1. Setup logging
    log := logger.FromEnv() // or logger.New(...)
    
    // 2. Create router
    r := chi.NewRouter()
    
    // 3. Add logging middleware
    r.Use(logger.ChiMiddleware(log))
    
    // 4. Add health check endpoint
    healthcheck.Mount(r)
    
    // 5. Setup protected routes with auth
    r.Route("/api", func(r chi.Router) {
        r.Use(auth.AuthMiddleware)
        
        r.Post("/lobbies", createLobbyHandler)
        r.Get("/lobbies/{id}", getLobbyHandler)
    })
}

func createLobbyHandler(w http.ResponseWriter, r *http.Request) {
    log := logger.Logger(r.Context())
    
    // Get authenticated user
    user, ok := auth.FromContext(r.Context())
    if !ok {
        httpx.WriteUnauthorized(w, "Authentication required", log)
        return
    }
    
    // Parse request
    var req CreateLobbyRequest
    if err := httpx.DecodeJSON(r, &req); err != nil {
        httpx.WriteBadRequest(w, "Invalid JSON body", 
            map[string]interface{}{"detail": err.Error()}, log)
        return
    }
    
    // Process request...
    httpx.WriteJSON(w, http.StatusCreated, lobby, log)
}
```

## Development Guidelines

### Adding New Libraries

When creating new libraries for the `libs` directory:

1. Follow the existing pattern and naming conventions
2. Include comprehensive README.md with examples
3. Add unit tests (`*_test.go` files)
4. Keep dependencies minimal (preferably only Go standard library)
5. Use proper Go module structure with `go.mod` and `go.sum`
6. Document the library's purpose and integration clearly

### Testing

All libraries include unit tests. Run tests for individual libraries:

```bash
cd backend/libs/auth && go test ./...
cd backend/libs/healthcheck && go test ./...
cd backend/libs/httpx && go test ./...
cd backend/libs/logger && go test ./...
```

### Versioning

These are internal libraries with no external versioning. They follow the monorepo pattern and should be used with `replace` directives in service modules.

## Library Reference

### auth
Authentication middleware that parses API Gateway headers (`X-User-ID`, `X-Username`) and provides a typed `User` struct for downstream handlers.

**Key types:**
- `User struct { ID uuid.UUID, Username string }`
- `AuthMiddleware` variable for default configuration

**Key functions:**
- `FromContext(ctx context.Context) (User, bool)`
- `NewAuthMiddleware(userHeader, usernameHeader string) func(http.Handler) http.Handler`

### healthcheck  
Simple health check endpoint that returns `200 OK` with plain text response `1`.

**Key functions:**
- `Mount(r chi.Router)` - Mount on chi router
- `Handler() http.Handler` - Get standalone handler

### httpx
HTTP response utilities providing standardized JSON responses and error handling.

**Key functions:**
- `WriteJSON(w, status, payload, log)` - Generic JSON response
- `WriteError(w, status, code, message, details, log)` - Error response
- `WriteBadRequest(w, message, details, log)` - 400 Bad Request
- `WriteUnauthorized(w, message, log)` - 401 Unauthorized
- `DecodeJSON(r, target)` - Decode request body

### logger
Structured logging library with JSON output and HTTP request middleware.

**Key types:**
- `Logger` (slog.Logger)

**Key functions:**
- `New(opts ...Option) *slog.Logger` - Create logger
- `FromEnv() *slog.Logger` - Configure from environment
- `ChiMiddleware(log *slog.Logger) func(http.Handler) http.Handler` - HTTP logging
- `Logger(ctx context.Context) *slog.Logger` - Get from context
- `WithLogger(ctx context.Context, log *slog.Logger) context.Context` - Set in context

## Contributing

When modifying these libraries:

1. Consider impact on all services that use the library
2. Update README documentation for any API changes  
3. Maintain backward compatibility when possible
4. Add or update tests for new functionality
5. Consider whether changes require updates to service dependencies

For questions or issues with these libraries, refer to the individual library READMEs or create an issue in the project repository.