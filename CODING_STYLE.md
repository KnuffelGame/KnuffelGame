# Coding Style Guide - KnuffelGame

This document outlines the coding conventions and best practices for the KnuffelGame project, incorporating patterns established in the SSEService implementation and serving as a binding reference for all future Go backend services.

## Table of Contents

1. [General Principles](#general-principles)
2. [Go Code Organization](#go-code-organization)
3. [Architecture Guidelines](#architecture-guidelines)
4. [Naming Conventions](#naming-conventions)
5. [Error Handling](#error-handling)
6. [Logging](#logging)
7. [HTTP Handlers](#http-handlers)
8. [Testing](#testing)
9. [Configuration](#configuration)
10. [Documentation](#documentation)
11. [Dependencies](#dependencies)
12. [Service Architecture Patterns](#service-architecture-patterns)
13. [Security Guidelines](#security-guidelines)

---

## General Principles

- **Simplicity over complexity**: Prefer straightforward, readable code over clever solutions
- **Explicit over implicit**: Make dependencies and behavior clear
- **Fail fast**: Validate early and return errors promptly
- **Composition over inheritance**: Use interfaces and dependency injection
- **Package-level globals with care**: Use `sync.Once` for safe lazy initialization when needed
- **OpenAPI-driven development**: Define API contracts first, implement to spec
- **Service-oriented architecture**: Design services for scalability and independent deployment

---

## Go Code Organization

### Enhanced Project Structure

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
        │   ├── router.go     # Route definitions and middleware composition
        │   ├── handlers/     # HTTP handlers
        │   ├── middleware/   # Custom middleware
        │   ├── models/       # Request/response models and domain logic
        │   ├── integration/  # Integration tests
        │   └── testutil/     # Testing utilities
        ├── pkg/              # Reusable packages
        │   └── config/       # Configuration loading
        ├── Dockerfile
        ├── go.mod
        ├── go.sum
        ├── openapi.yaml     # OpenAPI specification
        └── README.md
```

### Package Organization

- **`cmd/`**: Entry points only. Keep `main.go` minimal (< 60 lines)
- **`internal/`**: Service-specific implementation not exposed to other services
- **`pkg/`**: Reusable packages that could be shared (though prefer `libs/` for actual sharing)
- **`libs/`**: Standalone reusable libraries with their own `go.mod`
- **`internal/testutil/`**: Shared testing utilities for deterministic testing
- **`internal/integration/`**: Integration test suites

### File Naming

- Use lowercase with underscores for multi-word files: `create_token.go`, `validate_token.go`
- Test files: `<name>_test.go`
- One concept per file when appropriate (e.g., separate handler per file)
- Integration tests: `<name>_integration_test.go` in `integration/` directory

---

## Architecture Guidelines

### Service Bootstrap Patterns

Initialize services with consistent bootstrap sequence:

```go
func main() {
    // 1. Load configuration from environment with sensible defaults
    cfg := models.Load()
    
    // 2. Ensure SERVICE_NAME env is present
    if os.Getenv("SERVICE_NAME") == "" {
        _ = os.Setenv("SERVICE_NAME", cfg.ServiceName)
    }
    
    // 3. Initialize logger with configuration
    level := parseLevel(cfg.LogLevel)
    log := logger.New(
        logger.WithService(cfg.ServiceName),
        logger.WithLevel(level),
        logger.WithColor(cfg.LogColor),
    ).With(slog.String("component", "bootstrap"))
    
    // 4. Validate critical configuration
    if cfg.InternalToken == "" {
        log.Warn("SSE_INTERNAL_TOKEN is empty; internal endpoints will be blocked")
    }
    
    // 5. Build router with middleware
    r := router.New(cfg, log)
    
    // 6. HTTP server with timeouts
    srv := &http.Server{
        Addr:              ":" + cfg.Port,
        Handler:           r,
        ReadHeaderTimeout: 5 * time.Second,
        IdleTimeout:       120 * time.Second,
    }
    
    // 7. Start server with graceful shutdown
    go func() {
        log.Info("listening", slog.String("port", cfg.Port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Error("server exited", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }()
    
    // 8. Graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    <-stop
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    log.Info("shutting down")
    if err := srv.Shutdown(ctx); err != nil {
        log.Error("graceful shutdown failed", slog.String("error", err.Error()))
    }
}
```

### Middleware Composition Patterns

Compose middleware in logical order from outermost to innermost:

```go
func New(cfg *models.Config, log *slog.Logger) http.Handler {
    r := chi.NewRouter()
    
    // 1. Request correlation and logging (outermost)
    r.Use(chimw.RequestID)
    r.Use(logger.ChiMiddleware(log))
    
    // 2. CORS and rate limiting
    if mw := ssemw.CORS(cfg); mw != nil {
        r.Use(mw)
    }
    
    // 3. Healthcheck (bypass auth)
    healthcheck.Mount(r)
    
    // 4. Service-specific middleware
    r.Route("/events", func(r chi.Router) {
        // SSE-specific rate limiting
        if sseLimiter := ssemw.RateLimitSSE(cfg); sseLimiter != nil {
            r = r.With(sseLimiter)
        }
        
        // Auth and authorization chain
        r.Group(func(r chi.Router) {
            r.Use(ssemw.AuthMiddleware(cfg, log))
            r.Use(ssemw.AuthorizeLobbyMembership(cfg, log))
            r.Get("/lobby/{lobby_id}", handlers.SubscribeLobbyEvents(reg, cfg, log))
        })
    })
    
    // 5. Internal endpoints with additional protection
    r.Route("/internal", func(r chi.Router) {
        if internalGuard := ssemw.InternalOnly(cfg); internalGuard != nil {
            r.Use(internalGuard)
        }
        if internalLimiter := ssemw.RateLimitInternal(cfg); internalLimiter != nil {
            r.Use(internalLimiter)
        }
        r.Post("/publish", handlers.Publish(reg, log))
    })
    
    return r
}
```

### OpenAPI-Driven Development

- Define OpenAPI specs first: `openapi.yaml` in service root
- Generate models and validation from specs
- Test API conformance: validate responses against OpenAPI schema
- Document endpoint examples in OpenAPI spec

Example OpenAPI integration in handlers:

```go
// PublishEventRequest mirrors OpenAPI schema exactly
type PublishEventRequest struct {
    TargetType   string                 `json:"target_type"`              // "lobby" | "game"
    TargetID     string                 `json:"target_id"`                // lby_* or gam_*
    EventType    string                 `json:"event_type"`               // event name
    TargetUserID string                 `json:"target_user_id,omitempty"` // optional
    Data         map[string]interface{} `json:"data"`                     // payload
}

// Validate against OpenAPI schema requirements
func (req *PublishEventRequest) Validate() error {
    if req.TargetType != "lobby" && req.TargetType != "game" {
        return fmt.Errorf("target_type must be 'lobby' or 'game'")
    }
    if strings.TrimSpace(req.TargetID) == "" {
        return fmt.Errorf("target_id is required")
    }
    // ... additional validation
    return nil
}
```

### SSE/WebSocket Patterns

For Server-Sent Events implementation:

```go
// SSE Handler Pattern
func SubscribeLobbyEvents(reg *models.Registry, cfg *models.Config, baseLog *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Request-scoped logger
        log := requestLogger(r.Context(), baseLog).WithGroup("sse")
        
        // Validate required headers (from middleware)
        userID := r.Header.Get("X-User-ID")
        if userID == "" {
            Unauthorized(w, "Missing authentication", log)
            return
        }
        
        // Verify Flusher support for streaming
        flusher, ok := w.(http.Flusher)
        if !ok {
            InternalServerError(w, "Streaming unsupported", nil, log)
            return
        }
        
        // Set SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        w.Header().Set("X-Accel-Buffering", "no") // nginx
        
        // Initial flush
        writeRetryLine(w, 5000)
        flusher.Flush()
        
        // Register connection in registry
        _, conn := reg.RegisterConnection(models.TargetType("lobby"), lobbyID, userID)
        
        // Writer goroutine for event delivery
        go func() {
            for {
                select {
                case <-r.Context().Done():
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
        
        // Heartbeat management
        hbInterval := time.Duration(30) * time.Second
        if cfg != nil && cfg.HeartbeatInterval > 0 {
            hbInterval = cfg.HeartbeatInterval
        }
        ticker := time.NewTicker(hbInterval)
        defer ticker.Stop()
        
        // Stream loop with context cancellation
        for {
            select {
            case <-r.Context().Done():
                return
            case <-conn.Done:
                return
            case t := <-ticker.C:
                // Send heartbeat with backpressure handling
                payload := map[string]string{"timestamp": t.UTC().Format(time.RFC3339)}
                data, _ := json.Marshal(payload)
                select {
                case conn.Ch <- models.SSEMessage{Event: "keep_alive", Data: data}:
                default:
                    // Drop heartbeat if backpressure - cleanup will handle
                }
            }
        }
    }
}
```

### Service Integration Patterns

Call other services consistently:

```go
// Auth middleware calling APIGateway
func AuthMiddleware(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
    client := &http.Client{Timeout: 3 * time.Second}
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            log := logger.Logger(r.Context()).WithGroup("middleware")
            
            // Validate JWT via APIGateway
            validateURL := cfg.APIGatewayBaseURL + "/internal/validate"
            body, _ := json.Marshal(map[string]string{"token": c.Value})
            req, _ := http.NewRequest(http.MethodPost, validateURL, bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("X-Request-ID", rid) // propagate correlation
            
            resp, err := client.Do(req)
            if err != nil {
                log.Error("token validation failed", slog.String("error", err.Error()))
                handlers.Unauthorized(w, "Invalid token", log)
                return
            }
            defer resp.Body.Close()
            
            // Extract user context and propagate headers
            var v struct {
                Valid    bool   `json:"valid"`
                UserID   string `json:"user_id"`
                Username string `json:"username"`
            }
            _ = json.NewDecoder(resp.Body).Decode(&v)
            
            if v.Valid && v.UserID != "" {
                r.Header.Set("X-User-ID", v.UserID)
                r.Header.Set("X-Username", v.Username)
                ctx := context.WithValue(r.Context(), userCtxKey{}, UserContext{
                    UserID: v.UserID, Username: v.Username,
                })
                next.ServeHTTP(w, r.WithContext(ctx))
            } else {
                handlers.Unauthorized(w, "Invalid token", log)
            }
        })
    }
}
```

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
    Issuer         = "knuffel-auth-service"
    minSecretLen   = 32
    heartbeatInterval = 30 * time.Second
)
```

### Types

- **PascalCase** for all types (exported and unexported)
- Use descriptive names: `Generator`, `Validator`, `CreateJWTRequest`
- Struct field tags on same line:

```go
type PublishEventRequest struct {
    TargetType   string                 `json:"target_type"`
    TargetID     string                 `json:"target_id"`
    EventType    string                 `json:"event_type"`
    TargetUserID string                 `json:"target_user_id,omitempty"`
    Data         map[string]interface{} `json:"data"`
}
```

### Errors

- Prefix sentinel errors with `Err`: `ErrSecretMissing`, `ErrTokenExpired`, `ErrMalformedToken`
- Use `errors.New()` for simple errors
- Define errors at package level:

```go
var (
    ErrInvalidSignature   = errors.New("invalid signature")
    ErrTokenExpired       = errors.New("token expired")
    ErrMalformedToken     = errors.New("invalid format")
    ErrTargetNotFound     = errors.New("target entry not found")
    ErrConnectionStalled  = errors.New("connection stalled")
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

### OpenAPI-Compliant Error Responses

Use the standard `ErrorPayload` structure:

```go
type ErrorPayload struct {
    Error   string                 `json:"error"`      // Machine-readable code
    Message string                 `json:"message"`    // Human-readable description
    Details map[string]interface{} `json:"details,omitempty"` // Optional additional info
}
```

Error codes should be lowercase with underscores: `invalid_request`, `token_generation_failed`, `invalid_signature`, `target_not_found`, `connection_failed`.

### Centralized Error Handling

Use centralized error helpers in handlers package:

```go
// errors.go - central error mapping
package handlers

import (
    "net/http"
    "log/slog"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
)

func BadRequest(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
    httpx.WriteBadRequest(w, message, details, log)
}

func Unauthorized(w http.ResponseWriter, message string, log *slog.Logger) {
    httpx.WriteUnauthorized(w, message, log)
}

func Forbidden(w http.ResponseWriter, message string, log *slog.Logger) {
    httpx.WriteForbidden(w, message, log)
}

func NotFound(w http.ResponseWriter, message string, log *slog.Logger) {
    httpx.WriteNotFound(w, message, log)
}

func Conflict(w http.ResponseWriter, message string, log *slog.Logger) {
    httpx.WriteError(w, http.StatusConflict, "conflict", message, nil, log)
}

func InternalServerError(w http.ResponseWriter, message string, details map[string]interface{}, log *slog.Logger) {
    httpx.WriteInternalError(w, message, details, log)
}
```

### SSE-Specific Error Handling

For SSE endpoints, ensure proper headers before error responses:

```go
func sseErrorResponse(w http.ResponseWriter, status int, code, message string, log *slog.Logger) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    httpx.WriteError(w, status, code, message, nil, log)
}
```

### Service-to-Service Error Propagation

Propagate errors consistently between services:

```go
// Service call with error mapping
resp, err := client.Do(req)
if err != nil {
    log.Error("service call failed", slog.String("error", err.Error()))
    InternalServerError(w, "Service unavailable", nil, log)
    return
}
defer resp.Body.Close()

if resp.StatusCode == http.StatusForbidden {
    log.Info("service authorization failed", slog.Int("status", resp.StatusCode))
    Forbidden(w, "Access denied", log)
    return
}

if resp.StatusCode >= 500 {
    log.Error("service error", slog.Int("status", resp.StatusCode))
    InternalServerError(w, "Service error", nil, log)
    return
}
```

---

## Logging

### Logger Usage

Use structured logging with `log/slog`:

```go
import "log/slog"

log := logger.Logger(r.Context())
log.Info("connection established", 
    slog.String("user_id", userID),
    slog.String("target_type", "lobby"),
    slog.String("target_id", lobbyID),
)
```

### Log Levels

- **Debug**: Detailed information for troubleshooting
- **Info**: General informational messages (request completed, connections established)
- **Warn**: Warning messages that don't prevent operation (validation failed, rate limit approached)
- **Error**: Errors that prevent operation (service calls failed, authentication failed)

### Request-Scoped Logging

Create request-scoped loggers with correlation:

```go
func requestLogger(ctx context.Context, base *slog.Logger) *slog.Logger {
    l := logger.Logger(ctx)
    if l == nil {
        l = base
    }
    if l == nil {
        l = logger.Default()
    }
    return l.WithGroup("handler")
}
```

### SSE-Specific Logging

Log SSE lifecycle events:

```go
// Connection events
log.Info("client connected", 
    slog.String("target_type", tt),
    slog.String("target_id", id),
    slog.String("user_id", userID),
)

// Publish events
log.Info("publish completed",
    slog.String("target_type", tt),
    slog.String("target_id", req.TargetID),
    slog.String("event_type", req.EventType),
    slog.Int("connections_found", found),
    slog.Int("events_sent", sent),
    slog.Int("failed_connections", failed),
)

// Disconnect events
log.Info("client disconnected",
    slog.String("duration_ms", time.Since(connectTime).String()),
)
```

### Service Integration Logging

Log service-to-service interactions:

```go
log := logger.Logger(r.Context()).WithGroup("middleware").With(
    slog.String("action", "auth"),
    slog.String("service", "APIGateway"),
)

resp, err := client.Do(req)
if err != nil {
    log.Error("gateway call failed", slog.String("error", err.Error()))
    return
}
log.Info("authentication successful", 
    slog.String("user_id", v.UserID),
    slog.String("username", v.Username),
)
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
func Publish(reg *models.Registry, baseLog *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log := requestLogger(r.Context(), baseLog).WithGroup("internal")
        // Implementation
    }
}
```

### Request Processing Flow

1. **Get request-scoped logger**:
```go
log := requestLogger(r.Context(), baseLog).WithGroup("handler")
```

2. **Decode and validate request**:
```go
var req PublishEventRequest
if err := httpx.DecodeJSON(r, &req); err != nil {
    BadRequest(w, "Invalid JSON body", map[string]interface{}{"detail": err.Error()}, log)
    return
}
```

3. **Validate business logic**:
```go
tt := strings.ToLower(strings.TrimSpace(req.TargetType))
if tt != "lobby" && tt != "game" {
    BadRequest(w, "Invalid target_type", map[string]interface{}{"allowed": []string{"lobby", "game"}}, log)
    return
}
```

4. **Process with error handling**:
```go
dataBytes, err := json.Marshal(req.Data)
if err != nil {
    InternalServerError(w, "Failed to marshal payload", map[string]interface{}{"detail": err.Error()}, log)
    return
}
```

5. **Return response**:
```go
log.Info("publish completed", 
    slog.String("target_type", tt),
    slog.Int("events_sent", sent),
)
httpx.WriteJSON(w, http.StatusOK, resp, log)
```

### SSE Handler Patterns

SSE handlers require special consideration:

```go
func SubscribeLobbyEvents(reg *models.Registry, cfg *models.Config, baseLog *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Early validation
        userID := r.Header.Get("X-User-ID")
        if userID == "" {
            Unauthorized(w, "Missing authentication", log)
            return
        }
        
        // Verify streaming support
        flusher, ok := w.(http.Flusher)
        if !ok {
            InternalServerError(w, "Streaming unsupported", nil, log)
            return
        }
        
        // Set SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        
        // Immediate flush to establish stream
        writeRetryLine(w, 5000)
        flusher.Flush()
        
        // Register and manage connection...
    }
}
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

- Unit tests: One test file per source file: `create_token_test.go` for `create_token.go`
- Integration tests: `internal/integration/` directory
- Test utilities: `internal/testutil/` for shared testing patterns

### Test Structure

Use table-driven tests where appropriate:

```go
func TestAuthMiddleware(t *testing.T) {
    tests := []struct {
        name         string
        jwtValue     string
        wantStatus   int
        wantUserID   string
    }{
        {
            name:       "valid token",
            jwtValue:   "valid_token",
            wantStatus: http.StatusOK,
            wantUserID: "usr1",
        },
        {
            name:       "invalid token",
            jwtValue:   "invalid",
            wantStatus: http.StatusUnauthorized,
            wantUserID: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### SSE Testing Patterns

Test SSE streams with deterministic timing:

```go
// waitForEvent scans SSE stream until event observed
func waitForEvent(t *testing.T, r *bufio.Reader, event string, timeout time.Duration) (bool, string) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        line, err := r.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                return false, ""
            }
            if time.Now().Before(deadline) {
                time.Sleep(10 * time.Millisecond)
                continue
            }
            return false, ""
        }
        // Skip comments and retry directives
        if line == "\n" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "retry: ") {
            continue
        }
        if strings.HasPrefix(line, "event: ") {
            ev := strings.TrimSpace(strings.TrimPrefix(line, "event: "))
            var dataLine string
            if dl, err := r.ReadString('\n'); err == nil && strings.HasPrefix(dl, "data: ") {
                dataLine = strings.TrimSpace(strings.TrimPrefix(dl, "data: "))
            }
            if ev == event {
                return true, dataLine
            }
        }
    }
    return false, ""
}

// SSE client helper
func openSSE(t *testing.T, url string, jwtValue string, timeout time.Duration) (*http.Response, *bufio.Reader) {
    req, _ := http.NewRequest(http.MethodGet, url, nil)
    req.AddCookie(&http.Cookie{Name: "jwt", Value: jwtValue})
    client := &http.Client{Timeout: timeout}
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("open SSE failed: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        _ = resp.Body.Close()
        t.Fatalf("SSE status = %d, want 200", resp.StatusCode)
    }
    return resp, bufio.NewReader(resp.Body)
}
```

### Integration Testing

Create comprehensive integration tests with mock services:

```go
func TestIntegration_Lobby_MultiClient_BroadcastAndTargeted_Unregister(t *testing.T) {
    // Mock APIGateway
    gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch {
        case r.Method == http.MethodPost && r.URL.Path == "/internal/validate":
            // Mock validation logic
        case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/lobbies/"):
            // Mock membership check
        }
    }))
    t.Cleanup(gw.Close)

    // Test configuration with short heartbeats
    cfg := testutil.NewTestConfig()
    cfg.HeartbeatInterval = 100 * time.Millisecond
    
    // Run test scenarios
    resp1, r1 := openSSE(t, srv.URL+"/events/lobby/"+lobbyID, "t1", 2*time.Second)
    defer resp1.Body.Close()
    
    // Test broadcast and targeted messages
    publishEvent(t, "player_joined", lobbyID)
    if !waitForEvent(t, r1, "player_joined", 500*time.Millisecond) {
        t.Fatalf("client did not receive broadcast event")
    }
}
```

### Test Utility Patterns

Create deterministic test utilities:

```go
// NewTestConfig returns minimal config with short timeouts
func NewTestConfig() *models.Config {
    return &models.Config{
        Port:              "0",
        ServiceName:       "TestService",
        LogLevel:          "error",
        LogColor:          false,
        InternalToken:     "test-internal",
        HeartbeatInterval: 100 * time.Millisecond,
        CORSAllowOrigins:  []string{"http://localhost"},
        JWTCookieName:     "jwt",
    }
}

// NewTestLogger creates quiet logger for tests
func NewTestLogger() *slog.Logger {
    return logger.New(
        logger.WithLevel(slog.LevelError),
        logger.WithColor(false),
        logger.WithWriter(io.Discard),
    )
}
```

### Middleware Testing

Test middleware with proper context propagation:

```go
func TestAuthMiddleware_Success(t *testing.T) {
    cfg := testutil.NewTestConfig()
    gw := mockAPIGateway(t, cfg)
    
    r := testutil.NewTestRouter(cfg, testutil.NewTestLogger(), gw.URL)
    
    // Test authenticated request
    req := httptest.NewRequest(http.MethodGet, "/events/lobby/test", nil)
    req.AddCookie(&http.Cookie{Name: "jwt", Value: "valid_token"})
    
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
    
    // Verify headers were set
    if rec.Header().Get("X-User-ID") == "" {
        t.Fatal("X-User-ID header not set")
    }
}
```

### Mock Service Integration

Use httptest for mock services:

```go
func mockAPIGateway(t *testing.T, cfg *models.Config) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodPost:
            if r.URL.Path == "/internal/validate" {
                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(map[string]interface{}{
                    "valid":    true,
                    "user_id":  "test_user",
                    "username": "TestUser",
                })
                return
            }
        case http.MethodGet:
            if strings.HasPrefix(r.URL.Path, "/lobbies/") {
                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(map[string]interface{}{
                    "players": []map[string]string{{"user_id": "test_user"}},
                })
                return
            }
        }
        http.NotFound(w, r)
    }))
}
```

---

## Configuration

### Environment Variable Standards

Use SCREAMING_SNAKE_CASE consistently:

```go
// Common configuration pattern
type Config struct {
    // Server
    Port        string
    ServiceName string
    LogLevel    string
    LogColor    bool
    
    // CORS
    CORSAllowOrigins     []string
    CORSAllowMethods     []string
    CORSAllowHeaders     []string
    CORSAllowCredentials bool
    
    // Rate limiting
    RateLimitInternalPerMinute int
    RateLimitSSEPerMinute      int
    
    // Auth
    JWTCookieName string
    InternalToken string
    
    // Service-specific
    HeartbeatInterval time.Duration
    APIGatewayBaseURL string
}
```

### Configuration Loading

Load with sensible defaults:

```go
func Load() *Config {
    cfg := &Config{
        Port:        getenvDefault("PORT", "8084"),
        ServiceName: getenvDefault("SERVICE_NAME", "SSEService"),
        LogLevel:    getenvDefault("LOG_LEVEL", "info"),
        LogColor:    parseBoolEnv("LOG_COLOR", false),
        
        CORSAllowOrigins:     parseCSVEnv("CORS_ALLOW_ORIGINS", []string{"http://localhost:5173"}),
        CORSAllowMethods:     parseCSVEnv("CORS_ALLOW_METHODS", []string{"GET", "POST", "OPTIONS"}),
        CORSAllowHeaders:     parseCSVEnv("CORS_ALLOW_HEADERS", []string{"Accept", "Authorization", "Content-Type"}),
        CORSAllowCredentials: parseBoolEnv("CORS_ALLOW_CREDENTIALS", true),
        
        RateLimitInternalPerMinute: parseIntEnv("RATE_LIMIT_INTERNAL_PER_MINUTE", 60),
        RateLimitSSEPerMinute:      parseIntEnv("RATE_LIMIT_SSE_PER_MINUTE", 0),
        
        JWTCookieName: getenvDefault("JWT_COOKIE_NAME", "jwt"),
        InternalToken: os.Getenv("SSE_INTERNAL_TOKEN"),
        
        HeartbeatInterval: parseHeartbeatEnv("HEARTBEAT_INTERVAL_MS", 30_000),
        APIGatewayBaseURL: os.Getenv("APIGATEWAY_BASE_URL"),
    }
    return cfg
}
```

### CSV Environment Parsing

Handle list-based configuration:

```go
func parseCSVEnv(key string, def []string) []string {
    v := os.Getenv(key)
    if v == "" {
        return def
    }
    parts := strings.Split(v, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts {
        s := strings.TrimSpace(p)
        if s != "" {
            out = append(out, s)
        }
    }
    return out
}

func parseBoolEnv(key string, def bool) bool {
    v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
    switch v {
    case "1", "true", "yes", "on":
        return true
    case "0", "false", "no", "off":
        return false
    default:
        return def
    }
}
```

### Configuration Validation

Validate critical configuration early:

```go
func main() {
    cfg := config.Load()
    
    // Validate required configuration
    if cfg.JWTSecret == "" {
        log.Warn("JWT_SECRET is empty; token operations will fail")
    }
    
    if cfg.InternalToken == "" {
        log.Warn("SSE_INTERNAL_TOKEN is empty; internal endpoints will be blocked")
    }
    
    if cfg.APIGatewayBaseURL == "" {
        log.Warn("APIGATEWAY_BASE_URL is empty; membership checks will fail")
    }
}
```

### Environment Documentation

Document all configuration in `.env.example` files:

```bash
# Server
PORT=8084
SERVICE_NAME=SSEService
LOG_LEVEL=info
LOG_COLOR=false

# CORS
CORS_ALLOW_ORIGINS=http://localhost:5173,http://localhost:8080
CORS_ALLOW_METHODS=GET,POST,OPTIONS
CORS_ALLOW_HEADERS=Accept,Authorization,Content-Type,X-Requested-With,Cookie,X-Request-ID
CORS_ALLOW_CREDENTIALS=true

# Rate Limiting (0 = disabled)
RATE_LIMIT_INTERNAL_PER_MINUTE=60
RATE_LIMIT_SSE_PER_MINUTE=0

# Authentication
JWT_COOKIE_NAME=jwt
SSE_INTERNAL_TOKEN=change_me

# SSE Behavior
HEARTBEAT_INTERVAL_MS=30000

# Service Integration
APIGATEWAY_BASE_URL=http://localhost:8080
```

---

## Documentation

### Package Documentation

Every package should have a package comment:

```go
// Package httpx provides HTTP utility functions for JSON responses and error handling.
// It standardizes error responses across services with the ErrorPayload structure.
package httpx

// Package models provides data models and business logic for the SSEService.
// It includes connection registry management and SSE message handling.
package models

// Package testutil provides shared testing utilities for SSEService.
// It includes deterministic configuration and mock service helpers.
package testutil
```

### Function Documentation

Document exported functions with purpose, parameters, and behavior:

```go
// New constructs a new slog.Logger with the configured Options and sets it as the package default.
// Subsequent calls update the default. Safe for concurrent read but create early in startup.
func New(opts ...Option) *slog.Logger

// Mount registers a GET /healthcheck route on the provided chi router.
// The handler responds with status 200 and JSON {"status":"ok"}.
func Mount(r chi.Router)

// RegisterConnection creates channels and registers a connection for the given user on the target.
// Returns the target entry and the new connection.
func (r *Registry) RegisterConnection(tt TargetType, id, userID string) (*TargetEntry, *Connection)

// SubscribeLobbyEvents returns an SSE handler for lobby subscriptions.
// Validates lobby_id, requires X-User-ID header, registers connection, and manages heartbeat.
func SubscribeLobbyEvents(reg *Registry, cfg *Config, baseLog *slog.Logger) http.HandlerFunc
```

### Inline Comments

- Explain **why**, not **what** (code should be self-explanatory)
- Add context for non-obvious decisions:

```go
// Always emit guest tokens from this service
token, err := gen.CreateToken(req.UserID, req.Username, true)

// Drop heartbeat if backpressure; cleanup will be handled by writer or broadcast failures
select {
case conn.Ch <- msg:
default:
}

// Signal connection done (do not close conn.Ch to avoid send panics elsewhere)
select {
case conn.Done <- struct{}{}:
default:
}
```

### README Files

Each service should have comprehensive README with:

- **Purpose and architecture overview**
- **API documentation** with OpenAPI integration
- **Configuration requirements** with examples
- **Client examples** for external integrations
- **Deployment instructions** and environment setup

Example README structure (see SSEService/README.md):

```markdown
# Service Name

## Overview and Responsibilities
## Endpoints (per OpenAPI)
## Architecture and Design
## Security Model
## Configuration Model
## Logging and Observability
## Testing Strategy
## Build and Run
## Client Examples
```

### OpenAPI Integration

Integrate OpenAPI specifications:

```yaml
# Include detailed schemas, examples, and response documentation
# Link from README: "See [openapi.yaml](openapi.yaml) for detailed endpoints and schemas"
```

Document API conformance testing:

```go
// Test that handlers conform to OpenAPI specification
func TestPublish_Conformance(t *testing.T) {
    // Validate request/response against OpenAPI schema
    // Use generated client code from OpenAPI spec
}
```

---

## Dependencies

### Dependency Management

- Each library has its own `go.mod` (libs are independent)
- Services depend on local libs using replace directives in `go.mod`
- Use go modules: `go mod tidy` before committing

Example go.mod structure:

```go
module github.com/KnuffelGame/KnuffelGame/backend/services/SSEService

go 1.22.5

require (
    github.com/KnuffelGame/KnuffelGame/backend/libs/httpx v0.0.0
    github.com/KnuffelGame/KnuffelGame/backend/libs/logger v0.0.0
    github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck v0.0.0
    github.com/go-chi/chi/v5 v5.1.0
    github.com/go-chi/cors v1.2.1
    github.com/go-chi/httprate v0.7.4
)

replace github.com/KnuffelGame/KnuffelGame/backend/libs/httpx => ../../libs/httpx
replace github.com/KnuffelGame/KnuffelGame/backend/libs/logger => ../../libs/logger
replace github.com/KnuffelGame/KnuffelGame/backend/libs/healthcheck => ../../libs/healthcheck
```

### Standard Libraries

Prefer standard library when possible:
- `net/http` for HTTP handling
- `encoding/json` for JSON
- `log/slog` for structured logging
- `errors` for error handling
- `context` for request-scoped values
- `sync` for concurrency control

### External Dependencies

**Approved external libraries:**

- **github.com/go-chi/chi/v5**: HTTP router
- **github.com/go-chi/cors**: CORS handling
- **github.com/go-chi/httprate**: Rate limiting
- **github.com/golang-jwt/jwt/v5**: JWT tokens (AuthService)
- **github.com/go-playground/validator/v10**: Request validation
- **github.com/r3labs/sse**: Server-Sent Events engine (SSEService)

Add new dependencies with team consensus.

### Import Organization

Group imports in order:
1. Standard library
2. External dependencies  
3. Internal packages

```go
import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/cors"

    "github.com/KnuffelGame/KnuffelGame/backend/libs/httpx"
    "github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
    router "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal"
)
```

---

## Service Architecture Patterns

### Registry Patterns

Implement thread-safe registries for connection/state management:

```go
// Registry manages SSE targets and connections with thread safety
type Registry struct {
    mu      sync.RWMutex
    targets map[TargetKey]*TargetEntry
}

// Broadcast sends message to all or specific user's connection
func (r *Registry) Broadcast(tt TargetType, id string, msg SSEMessage, targetUserID string) (found int, sent int, failed int, ok bool) {
    key := MakeTargetKey(tt, id)
    
    // Collect recipients under read lock
    r.mu.RLock()
    te, exists := r.targets[key]
    if !exists {
        r.mu.RUnlock()
        return 0, 0, 0, false
    }
    
    var recipients []*Connection
    if targetUserID != "" {
        if c, has := te.Connections[targetUserID]; has {
            recipients = append(recipients, c)
        }
    } else {
        for _, c := range te.Connections {
            recipients = append(recipients, c)
        }
    }
    r.mu.RUnlock()
    
    // Dispatch with non-blocking sends
    for _, c := range recipients {
        select {
        case c.Ch <- msg:
            sent++
        default:
            failed++
            go r.RemoveConnection(tt, id, c.UserID) // cleanup stalled
        }
    }
    
    return len(recipients), sent, failed, true
}
```

### SSE Connection Lifecycle

Implement robust connection lifecycle management:

```go
// Connection represents a single client connection
type Connection struct {
    UserID string
    Ch     chan SSEMessage    // Buffered channel for backpressure
    Done   chan struct{}      // Signal completion
}

// RegisterConnection creates and registers connection
func (r *Registry) RegisterConnection(tt TargetType, id, userID string) (*TargetEntry, *Connection) {
    te := r.EnsureTarget(tt, id)
    
    conn := &Connection{
        UserID: userID,
        Ch:     make(chan SSEMessage, 16), // Buffer for burst handling
        Done:   make(chan struct{}, 1),    // Signal-only channel
    }
    
    r.mu.Lock()
    te.Connections[userID] = conn
    r.mu.Unlock()
    
    return te, conn
}

// RemoveConnection safely removes and signals completion
func (r *Registry) RemoveConnection(tt TargetType, id, userID string) {
    key := MakeTargetKey(tt, id)
    
    r.mu.Lock()
    te, ok := r.targets[key]
    if !ok {
        r.mu.Unlock()
        return
    }
    conn, exists := te.Connections[userID]
    if exists {
        delete(te.Connections, userID)
    }
    r.mu.Unlock()
    
    // Signal completion without closing channels
    if exists && conn != nil {
        select {
        case conn.Done <- struct{}{}:
        default:
        }
    }
}
```

### Service Mesh Patterns

Implement internal-only endpoints with shared secret auth:

```go
// InternalOnly middleware protects internal endpoints
func InternalOnly(cfg *models.Config) func(http.Handler) http.Handler {
    if cfg == nil || cfg.InternalToken == "" {
        return nil
    }
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("X-Internal-Token")
            if token == "" || token != cfg.InternalToken {
                handlers.Unauthorized(w, "Invalid or missing internal token", log)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Health Monitoring

Expose comprehensive health checks:

```go
// GetConnectionStats returns registry statistics
func GetConnectionStats(reg *models.Registry) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log := logger.Logger(r.Context())
        
        totalTargets, totalConnections, lobbyCount, lobbyConnections, gameCount, gameConnections := reg.Stats()
        
        stats := map[string]interface{}{
            "total_targets":        totalTargets,
            "total_connections":    totalConnections,
            "lobbies": map[string]interface{}{
                "targets":     lobbyCount,
                "connections": lobbyConnections,
            },
            "games": map[string]interface{}{
                "targets":     gameCount,
                "connections": gameConnections,
            },
            "timestamp": time.Now().UTC().Format(time.RFC3339),
        }
        
        httpx.WriteJSON(w, http.StatusOK, stats, log)
    }
}
```

---

## Security Guidelines

### JWT Validation Patterns

Validate JWT tokens consistently across services:

```go
// Auth middleware with JWT validation via AuthService
func AuthMiddleware(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
    client := &http.Client{Timeout: 3 * time.Second}
    cookieName := cfg.JWTCookieName
    validateURL := cfg.APIGatewayBaseURL + "/internal/validate"
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract JWT from cookie
            c, err := r.Cookie(cookieName)
            if err != nil || c == nil || c.Value == "" {
                handlers.Unauthorized(w, "Invalid or expired authentication token", log)
                return
            }
            
            // Validate via AuthService
            body, _ := json.Marshal(map[string]string{"token": c.Value})
            req, _ := http.NewRequest(http.MethodPost, validateURL, bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("X-Internal-Token", cfg.InternalToken)
            
            resp, err := client.Do(req)
            if err != nil {
                handlers.Unauthorized(w, "Invalid or expired authentication token", log)
                return
            }
            defer resp.Body.Close()
            
            var v struct {
                Valid    bool   `json:"valid"`
                UserID   string `json:"user_id"`
                Username string `json:"username"`
            }
            if err := json.NewDecoder(resp.Body).Decode(&v); err != nil || !v.Valid || v.UserID == "" {
                handlers.Unauthorized(w, "Invalid or expired authentication token", log)
                return
            }
            
            // Propagate user context
            r.Header.Set("X-User-ID", v.UserID)
            r.Header.Set("X-Username", v.Username)
            ctx := context.WithValue(r.Context(), userCtxKey{}, UserContext{
                UserID: v.UserID, Username: v.Username,
            })
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Membership Authorization

Authorize resource access via service calls:

```go
// AuthorizeLobbyMembership checks lobby membership via APIGateway
func AuthorizeLobbyMembership(cfg *models.Config, baseLog *slog.Logger) func(next http.Handler) http.Handler {
    client := &http.Client{Timeout: 3 * time.Second}
    cookieName := cfg.JWTCookieName
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := r.Header.Get("X-User-ID")
            if userID == "" {
                handlers.Unauthorized(w, "Invalid authentication", log)
                return
            }
            
            // Get lobby membership from APIGateway
            lobbyID := chi.URLParam(r, "lobby_id")
            req, _ := http.NewRequest(http.MethodGet, cfg.APIGatewayBaseURL+"/lobbies/"+lobbyID, nil)
            
            // Forward JWT cookie for gateway authorization
            if c, err := r.Cookie(cookieName); err == nil && c != nil {
                req.Header.Set("Cookie", cookieName+"="+c.Value)
            }
            
            resp, err := client.Do(req)
            if err != nil {
                handlers.InternalServerError(w, "Service unavailable", nil, log)
                return
            }
            defer resp.Body.Close()
            
            switch resp.StatusCode {
            case http.StatusOK:
                // Verify user is in players list
                var body struct {
                    Players []struct{ UserID string } `json:"players"`
                }
                if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
                    found := false
                    for _, p := range body.Players {
                        if p.UserID == userID {
                            found = true
                            break
                        }
                    }
                    if !found {
                        handlers.Forbidden(w, "You are not a member of this lobby", log)
                        return
                    }
                }
                next.ServeHTTP(w, r)
            case http.StatusForbidden:
                handlers.Forbidden(w, "You are not a member of this lobby", log)
            case http.StatusNotFound:
                handlers.NotFound(w, "Lobby not found", log)
            default:
                handlers.InternalServerError(w, "Service error", nil, log)
            }
        })
    }
}
```

### Internal-Only Endpoint Protection

Protect administrative endpoints:

```go
// InternalOnly protects endpoints with shared secret
func InternalOnly(cfg *models.Config) func(http.Handler) http.Handler {
    if cfg == nil || cfg.InternalToken == "" {
        return func(next http.Handler) http.Handler {
            return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                handlers.Unauthorized(w, "Internal endpoints disabled (no token configured)", log)
            })
        }
    }
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("X-Internal-Token")
            if token == "" || token != cfg.InternalToken {
                handlers.Unauthorized(w, "Invalid or missing internal token", log)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### CORS Configuration

Configure CORS for web clients:

```go
// CORS middleware for web clients
func CORS(cfg *models.Config) func(http.Handler) http.Handler {
    return cors.Handler(cors.Options{
        AllowedOrigins:   cfg.CORSAllowOrigins,
        AllowedMethods:   cfg.CORSAllowMethods,
        AllowedHeaders:   cfg.CORSAllowHeaders,
        AllowCredentials: cfg.CORSAllowCredentials,
        MaxAge:           300, // 5 minutes
    })
}
```

### Rate Limiting Strategies

Implement different rate limits per endpoint type:

```go
// Rate limiting for internal endpoints
func RateLimitInternal(cfg *models.Config) func(http.Handler) http.Handler {
    if cfg == nil || cfg.RateLimitInternalPerMinute <= 0 {
        return nil
    }
    return httprate.LimitByIP(cfg.RateLimitInternalPerMinute, 1*time.Minute)
}

// Rate limiting for SSE endpoints (usually disabled)
func RateLimitSSE(cfg *models.Config) func(http.Handler) http.Handler {
    if cfg == nil || cfg.RateLimitSSEPerMinute == 0 {
        return nil // Disabled for long-lived connections
    }
    if cfg.RateLimitSSEPerMinute < 0 {
        return nil
    }
    return httprate.LimitByIP(cfg.RateLimitSSEPerMinute, 1*time.Minute)
}
```

---

## Additional Conventions

### Context Usage

- Always pass `context.Context` as first parameter: `func Process(ctx context.Context, ...)`
- Use context for request-scoped values (logger, user info)
- Check context cancellation for long-running operations:

```go
// Long-running operation with context
func processEvents(ctx context.Context, reg *models.Registry) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Process events
        }
    }
}
```

### Concurrency

Use appropriate synchronization primitives:

```go
// RWMutex for read-heavy workloads
type Registry struct {
    mu      sync.RWMutex
    targets map[TargetKey]*TargetEntry
}

// Buffered channels for backpressure handling
conn := &Connection{
    Ch:   make(chan SSEMessage, 16), // Prevent blocking on writes
    Done: make(chan struct{}, 1),    // Signal-only
}

// Non-blocking sends with timeout handling
select {
case msg := <-conn.Ch:
    // Process message
case <-time.After(timeout):
    // Handle timeout
case <-ctx.Done():
    return
}
```

### Options Pattern

Use functional options for flexible configuration:

```go
type Options struct {
    Level   slog.Level
    Service string
    Color   bool
    Writer  io.Writer
}

type Option func(*Options)

func WithLevel(l slog.Level) Option {
    return func(o *Options) { o.Level = l }
}

func New(opts ...Option) *slog.Logger {
    o := &Options{
        Level:  slog.LevelInfo,
        Service: "service",
        Color:  true,
        Writer: os.Stdout,
    }
    
    for _, opt := range opts {
        opt(o)
    }
    
    return slog.New(slog.NewTextHandler(o.Writer, &slog.HandlerOptions{
        Level: o.Level,
        AddSource: true,
    }))
}
```

### Middleware Standards

Create composable middleware:

```go
// Standard middleware signature
type Middleware func(next http.Handler) http.Handler

// Middleware with dependencies
func AuthMiddleware(cfg *Config, log *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Implementation
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Docker Conventions

### Dockerfile Structure

Use multi-stage builds with security considerations:

```dockerfile
FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN go build -o sse-service ./cmd/SSEService

FROM alpine:3.22.2

WORKDIR /app
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

COPY --from=builder /app/sse-service .

# Healthcheck with proper interval
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8084/healthcheck || exit 1

USER appuser
EXPOSE 8084

CMD ["./sse-service"]
```

### Healthcheck Standards

- All services must expose `/healthcheck` endpoint
- Use the `healthcheck` library: `healthcheck.Mount(r)`
- Returns JSON: `{"status":"ok"}` with 200 status
- Include service-specific health checks where relevant

### Docker Compose Integration

Define service dependencies and networking:

```yaml
services:
  sse-service:
    build: ./backend/services/SSEService
    ports:
      - "8084:8084"
    environment:
      - SERVICE_NAME=SSEService
      - LOG_LEVEL=info
      - APIGATEWAY_BASE_URL=http://api-gateway:8080
      - SSE_INTERNAL_TOKEN=${SSE_INTERNAL_TOKEN}
    depends_on:
      - api-gateway
    networks:
      - backend
```

---

## Summary

This enhanced coding style guide establishes comprehensive standards for Go backend services, incorporating patterns from the SSEService implementation. Key principles:

1. **Consistency**: Unified patterns across all services
2. **Reliability**: Robust error handling and logging
3. **Scalability**: Service-oriented architecture with proper resource management
4. **Security**: Comprehensive authentication, authorization, and protection mechanisms
5. **Testability**: Deterministic testing with proper mocks and utilities
6. **Maintainability**: Clear documentation and coding conventions

### Quick Reference

- **Always use OpenAPI-driven development**
- **Implement proper middleware composition** (logger → auth → handler)
- **Use centralized error handling** with httpx package
- **Follow structured logging** with request correlation
- **Create comprehensive tests** including integration tests
- **Document configuration** in .env.example files
- **Validate security** on all endpoints
- **Monitor health** via standardized endpoints

When implementing new services, adapt these patterns and maintain consistency with existing implementations. The SSEService serves as the reference implementation for all future backend services.

Happy coding! 🧸
