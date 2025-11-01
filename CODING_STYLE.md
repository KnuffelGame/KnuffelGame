# Coding Style Guide - KnuffelGame

This document outlines the coding conventions and best practices for the KnuffelGame project based on the existing backend libraries and services.

## Table of Contents

1. [General Principles](#general-principles)
2. [Go Code Organization](#go-code-organization)
3. [Naming Conventions](#naming-conventions)
4. [Error Handling](#error-handling)
5. [Logging](#logging)
6. [HTTP Handlers](#http-handlers)
7. [Testing](#testing)
8. [Configuration](#configuration)
9. [Documentation](#documentation)
10. [Dependencies](#dependencies)

---

## General Principles

- **Simplicity over complexity**: Prefer straightforward, readable code over clever solutions
- **Explicit over implicit**: Make dependencies and behavior clear
- **Fail fast**: Validate early and return errors promptly
- **Composition over inheritance**: Use interfaces and dependency injection
- **Package-level globals with care**: Use `sync.Once` for safe lazy initialization when needed

---

## Go Code Organization

### Project Structure

```
backend/
├── libs/                    # Shared libraries
│   └── <package>/
│       ├── <package>.go     # Main implementation
│       ├── <package>_test.go
│       ├── go.mod
│       ├── go.sum
│       └── README.md
└── services/
    └── <ServiceName>/
        ├── cmd/
        │   └── <ServiceName>/
        │       └── main.go   # Entry point
        ├── internal/         # Service-specific code
        │   ├── router.go     # Route definitions
        │   ├── handlers/     # HTTP handlers
        │   ├── middleware/   # Custom middleware
        │   ├── models/       # Request/response models
        │   └── <domain>/     # Domain logic (e.g., jwt/)
        ├── pkg/
        │   └── config/       # Configuration loading
        │       └── config.go
        ├── Dockerfile
        ├── go.mod
        ├── go.sum
        └── README.md
```

### Package Organization

- **`cmd/`**: Entry points only. Keep `main.go` minimal (< 50 lines)
- **`internal/`**: Service-specific implementation not exposed to other services
- **`pkg/`**: Reusable packages that could be shared (though prefer `libs/` for actual sharing)
- **`libs/`**: Standalone reusable libraries with their own `go.mod`

### File Naming

- Use lowercase with underscores for multi-word files: `create_token.go`, `validate_token.go`
- Test files: `<name>_test.go`
- One concept per file when appropriate (e.g., separate handler per file)

---

## Naming Conventions

### Variables and Functions

- **camelCase** for private: `defaultLogger`, `parseLevel`, `remoteIP`
- **PascalCase** for exported: `New`, `Default`, `WriteJSON`, `ChiMiddleware`
- Use descriptive names: `requestID` over `rid`, `validator` over `v`
- Single-letter variables acceptable for:
  - Loop indices: `i`, `j`
  - Short-lived function params in obvious contexts: `r *http.Request`, `w http.ResponseWriter`
  - Receivers: `(g *Generator)`, `(v *Validator)`

### Constants

- **PascalCase** for exported: `Issuer`, `ErrSecretMissing`
- **camelCase** for private: `maxBodySize`, `minSecretLen`
- Group related constants together

```go
const (
    Issuer       = "knuffel-auth-service"
    minSecretLen = 32
)
```

### Types

- **PascalCase** for all types (exported and unexported)
- Use descriptive names: `Generator`, `Validator`, `CreateJWTRequest`
- Struct field tags on same line:

```go
type CreateJWTRequest struct {
    UserID   string `json:"user_id" validate:"required,uuid4"`
    Username string `json:"username" validate:"required,min=3,max=20,usernameFmt"`
}
```

### Errors

- Prefix sentinel errors with `Err`: `ErrSecretMissing`, `ErrTokenExpired`, `ErrMalformedToken`
- Use `errors.New()` for simple errors
- Define errors at package level:

```go
var (
    ErrInvalidSignature = errors.New("invalid signature")
    ErrTokenExpired     = errors.New("token expired")
    ErrMalformedToken   = errors.New("invalid format")
)
```

---

## Error Handling

### General Rules

- **Check all errors**: Never ignore errors silently
- **Return early**: Avoid deep nesting with guard clauses
- **Log before returning**: Add context to errors via logging

```go
if err := dec.Decode(&req); err != nil {
    log.Warn("decode failed", slog.String("error", err.Error()))
    httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", 
        map[string]interface{}{"detail": err.Error()}, log)
    return
}
```

### Error Responses

Use the standard `ErrorPayload` structure:

```go
type ErrorPayload struct {
    Error   string                 `json:"error"`      // Machine-readable code
    Message string                 `json:"message"`    // Human-readable description
    Details map[string]interface{} `json:"details,omitempty"` // Optional additional info
}
```

Error codes should be lowercase with underscores: `invalid_request`, `token_generation_failed`, `invalid_signature`

### Error Mapping

Translate internal errors to user-friendly responses:

```go
func mapValidationError(err error) string {
    switch err {
    case jwt.ErrInvalidSignature:
        return "invalid signature"
    case jwt.ErrTokenExpired:
        return "token expired"
    default:
        return "invalid format"
    }
}
```

---

## Logging

### Logger Usage

Use structured logging with `log/slog`:

```go
import "log/slog"

log := logger.Logger(r.Context())
log.Info("token generated", 
    slog.String("user_id", userID),
    slog.String("username", username),
    slog.Bool("guest", true))
```

### Log Levels

- **Debug**: Detailed information for troubleshooting (e.g., token created/validated)
- **Info**: General informational messages (e.g., request completed, token generated)
- **Warn**: Warning messages that don't prevent operation (e.g., decode failed, validation failed, weak secret)
- **Error**: Errors that prevent operation (e.g., token generation failed, server exited)

### Log Attributes

- Use descriptive attribute keys: `user_id`, `error`, `component`, `action`
- Group related attributes:

```go
log := logger.Default().WithGroup("jwt").With(slog.String("component", "generator"))
```

- In handlers, create scoped loggers:

```go
log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "create_token"))
```

### Environment Configuration

Use `logger.FromEnv()` in services:

```go
// Reads: LOG_LEVEL, SERVICE_NAME, LOG_COLOR
log := logger.FromEnv().With(slog.String("component", "bootstrap"))
```

---

## HTTP Handlers

### Handler Structure

Use factory functions returning `http.HandlerFunc` with dependency injection:

```go
func CreateTokenHandler(gen *jwt.Generator) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation
    }
}
```

### Request Processing Flow

1. **Limit body size**:
```go
r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
```

2. **Get logger from context**:
```go
log := logger.Logger(r.Context()).WithGroup("handler").With(slog.String("action", "create_token"))
```

3. **Decode request**:
```go
var req models.CreateJWTRequest
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&req); err != nil {
    log.Warn("decode failed", slog.String("error", err.Error()))
    httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", 
        map[string]interface{}{"detail": err.Error()}, log)
    return
}
```

4. **Validate**:
```go
errMap := req.Validate()
if len(errMap) > 0 {
    log.Info("validation failed", slog.Any("errors", errMap))
    httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Validation failed", 
        map[string]interface{}{"fields": errMap}, log)
    return
}
```

5. **Process business logic**:
```go
token, err := gen.CreateToken(req.UserID, req.Username, true)
if err != nil {
    log.Error("token generation failed", slog.String("error", err.Error()))
    httpx.WriteError(w, http.StatusInternalServerError, "token_generation_failed", 
        "Failed to generate JWT token", map[string]interface{}{"detail": err.Error()}, log)
    return
}
```

6. **Return response**:
```go
log.Info("token generated", slog.String("user_id", req.UserID))
httpx.WriteJSON(w, http.StatusOK, models.CreateTokenResponse{Token: token}, log)
```

### Response Helpers

Use `httpx` package helpers for consistency:

- `WriteJSON(w, status, payload, log)` - Generic JSON response
- `WriteError(w, status, code, message, details, log)` - Error response
- `WriteBadRequest(w, message, details, log)` - 400 errors
- `WriteUnauthorized(w, message, log)` - 401 errors
- `WriteForbidden(w, message, log)` - 403 errors
- `WriteNotFound(w, message, log)` - 404 errors
- `WriteInternalError(w, message, details, log)` - 500 errors
- `WriteNoContent(w)` - 204 responses

---

## Testing

### Test File Organization

- One test file per source file: `create_token_test.go` for `create_token.go`
- Test function naming: `Test<FunctionName>_<Scenario>`

```go
func TestCreateToken_Success(t *testing.T) { }
func TestCreateToken_MissingUserID(t *testing.T) { }
func TestCreateToken_ValidationFail(t *testing.T) { }
```

### Test Structure

Use table-driven tests where appropriate:

```go
func TestWriteJSON(t *testing.T) {
    tests := []struct {
        name       string
        status     int
        payload    interface{}
        wantStatus int
        wantBody   string
    }{
        {
            name:       "success with map",
            status:     http.StatusOK,
            payload:    map[string]string{"message": "hello"},
            wantStatus: http.StatusOK,
            wantBody:   `{"message":"hello"}`,
        },
        // More cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### HTTP Handler Testing

Use `httptest` for handler testing:

```go
func TestCreateToken_Success(t *testing.T) {
    gen := jwt.NewGenerator("12345678901234567890123456789012")
    h := CreateTokenHandler(gen)
    body, _ := json.Marshal(map[string]interface{}{
        "username": "Alice",
        "user_id": "550e8400-e29b-41d4-a716-446655440000",
    })
    req := httptest.NewRequest(http.MethodPost, "/internal/create", bytes.NewReader(body))
    rec := httptest.NewRecorder()
    
    h(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
}
```

### Test Dependencies

- Use real implementations when lightweight (avoid over-mocking)
- Use test secrets/keys: minimum 32 characters for JWT tests
- Use `bytes.Buffer` for logger output testing

---

## Configuration

### Environment Variables

Load configuration from environment in a dedicated `config` package:

```go
package config

import "os"

type Config struct {
    JWTSecret string
    Port      string
}

func Load() *Config {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"  // Sensible default
    }
    return &Config{
        JWTSecret: os.Getenv("JWT_SECRET"),
        Port:      port,
    }
}
```

### Configuration Validation

Validate critical configuration early in `main()`:

```go
cfg := config.Load()
if cfg.JWTSecret == "" {
    log.Warn("JWT_SECRET is empty; token operations will fail")
}
```

### Environment Variable Naming

- Use SCREAMING_SNAKE_CASE: `JWT_SECRET`, `LOG_LEVEL`, `SERVICE_NAME`
- Be consistent across services
- Document in `.env.example` files

---

## Documentation

### Package Documentation

Every package should have a package comment:

```go
// Package httpx provides HTTP utility functions for JSON responses and error handling.
// It standardizes error responses across services with the ErrorPayload structure.
package httpx
```

### Function Documentation

Document exported functions with purpose, parameters, and behavior:

```go
// New constructs a new slog.Logger with the configured Options and sets it as the package default.
// Subsequent calls update the default. Safe for concurrent read but create early in startup.
func New(opts ...Option) *slog.Logger { }

// Mount registers a GET /healthcheck route on the provided chi router.
// The handler responds with status 200 and JSON {"status":"ok"}.
func Mount(r chi.Router) { }
```

### Inline Comments

- Explain **why**, not **what** (code should be self-explanatory)
- Add context for non-obvious decisions:

```go
// Always guest tokens from this service
token, err := gen.CreateToken(req.UserID, req.Username, true)
```

### README Files

Each library and service should have a README with:
- Purpose and overview
- Usage examples
- Configuration requirements
- API documentation (for services)

---

## Dependencies

### Dependency Management

- Each library has its own `go.mod` (libs are independent)
- Services depend on local libs using replace directives in `go.mod`
- Use go modules: `go mod tidy` before committing

### Standard Libraries

Prefer standard library when possible:
- `net/http` for HTTP handling
- `encoding/json` for JSON
- `log/slog` for structured logging
- `errors` for error handling

### External Dependencies

Approved external libraries:
- **github.com/go-chi/chi/v5**: HTTP router
- **github.com/golang-jwt/jwt/v5**: JWT tokens
- **github.com/go-playground/validator/v10**: Request validation

Add new dependencies with team consensus.

### Import Organization

Group imports in order:
1. Standard library
2. External dependencies  
3. Internal packages

```go
import (
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/go-chi/chi/v5"

    "github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
)
```

---

## Additional Conventions

### Context Usage

- Always pass `context.Context` as first parameter: `func Process(ctx context.Context, ...)`
- Use context for request-scoped values (e.g., logger)
- Check context cancellation for long-running operations

### Concurrency

- Use `sync.Once` for one-time initialization:

```go
var defaultLogger *slog.Logger
var defaultOnce sync.Once

func Default() *slog.Logger {
    defaultOnce.Do(func() {
        if defaultLogger == nil {
            defaultLogger = New()
        }
    })
    return defaultLogger
}
```

### Options Pattern

Use functional options for flexible configuration:

```go
type Options struct {
    Level   slog.Level
    Service string
}

type Option func(*Options)

func WithLevel(l slog.Level) Option { 
    return func(o *Options) { o.Level = l } 
}

func New(opts ...Option) *slog.Logger {
    o := applyOptions(opts)
    // Use options...
}
```

### Middleware

- Middleware should be composable
- Inject dependencies via closure:

```go
func ChiMiddleware(l *slog.Logger) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Middleware logic
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Docker Conventions

### Dockerfile Structure

Use multi-stage builds:

```dockerfile
FROM golang:1.25.3-alpine AS builder

COPY . /app
WORKDIR /app/services/<ServiceName>
RUN go build -o <service> ./cmd/<ServiceName>

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add curl
COPY --from=builder /app/services/<ServiceName>/<service> .

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:<PORT>/healthcheck || exit 1

EXPOSE <PORT>

CMD ["./<service>"]
```

### Healthcheck

- All services must expose `/healthcheck` endpoint
- Use the `healthcheck` library: `healthcheck.Mount(r)`
- Returns JSON: `{"status":"ok"}` with 200 status

---

## Summary

This style guide ensures consistency, maintainability, and quality across the KnuffelGame backend codebase. When in doubt:

1. **Look at existing code** in `libs/` and `services/AuthService/`
2. **Keep it simple** and readable
3. **Ask the team** for clarification

Happy coding! 🧸

