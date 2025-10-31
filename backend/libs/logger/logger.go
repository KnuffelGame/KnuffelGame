package logger

import (
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

// New constructs a new slog.Logger with the configured Options and sets it as the package default.
// Subsequent calls update the default. Safe for concurrent read but create early in startup.
func New(opts ...Option) *slog.Logger {
	o := applyOptions(opts)
	h := newColorJSONHandler(o.Writer, o)
	l := slog.New(h)
	defaultLogger = l
	return l
}

// Default returns the package global logger; if nil initializes with defaults.
func Default() *slog.Logger {
	if defaultLogger == nil {
		return New()
	}
	return defaultLogger
}

// FromEnv builds a logger using environment variables:
// LOG_LEVEL: debug|info|warn|error (default: info)
// SERVICE_NAME: string (optional)
// LOG_COLOR: 1|true enables color; 0|false disables (default: off)
func FromEnv() *slog.Logger {
	lvl := parseLevel(os.Getenv("LOG_LEVEL"))
	service := os.Getenv("SERVICE_NAME")
	color := parseBool(os.Getenv("LOG_COLOR"))
	return New(WithLevel(lvl), WithService(service), WithColor(color))
}

func parseLevel(s string) slog.Level {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}
