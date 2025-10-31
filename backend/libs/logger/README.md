# logger

Opinionated slog setup for KnuffelGame services.

Features:
- JSON structured logs
- Optional ANSI color per level (Debug blue, Info green, Warn yellow, Error red)
- Service name attribution
- Context helpers (`logger.Logger(ctx)` / `logger.WithLogger(ctx, log)`) for application logic
- Chi middleware for HTTP request logging with latency, status, request id and more
- Simple Option pattern for configuration

## Install

Add the module to your service (from a service module root):

```bash
go get github.com/KnuffelGame/KnuffelGame/backend/libs/logger
```

## Quick start

```go
import (
  slog "log/slog"
  "github.com/go-chi/chi/v5"
  "github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
)

func main() {
  log := logger.New(
    logger.WithService("AuthService"),
    logger.WithLevel(slog.LevelInfo),
    logger.WithColor(true),
  )

  r := chi.NewRouter()
  r.Use(logger.ChiMiddleware(log))

  r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
    logger.Logger(r.Context()).Info("handler invoked")
    w.Write([]byte("hi"))
  })

  http.ListenAndServe(":8080", r)
}
```

## Environment helper

`FromEnv()` reads environment variables to configure the logger:

| Variable     | Values                                                  | Default    | Description                                        |
|--------------|---------------------------------------------------------|------------|----------------------------------------------------|
| LOG_LEVEL    | debug, info, warn, error                                | info       | Minimum log level emitted.                         |
| SERVICE_NAME | any string                                              | "" (empty) | Service identifier added as `service` attribute.   |
| LOG_COLOR    | 1, true, yes, on (enable) / 0, false, no, off (disable) | disabled   | Wraps each JSON line in ANSI color based on level. |

Usage:
```go
log := logger.FromEnv()
```

## Chi request logging
Logs once per request on completion (level INFO) with attributes:
- http.method
- http.path
- http.status
- http.duration_ms
- http.request_id (generated if missing)
- http.remote_ip
- http.user_agent

## Options
- `WithLevel(level slog.Level)`
- `WithService(name string)`
- `WithColor(enabled bool)`
- `WithAddSource(enabled bool)` include source file:line
- `WithWriter(w io.Writer)` custom destination

## Context helpers
- `Logger(ctx)` returns *slog.Logger (falls back to default global)
- `WithLogger(ctx, l)` attaches logger to context

## Global default
Calling `logger.New()` also sets / updates the package global returned by `logger.Default()`.

## Notes on color mode
When `LOG_COLOR` (or `WithColor(true)`) is enabled, the JSON line is wrapped with ANSI escape codes. Some log collectors may need to strip these codes to parse JSON correctly.

## License
MIT (inherited project policy)
