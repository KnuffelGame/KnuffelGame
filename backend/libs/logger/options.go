package logger

import (
	"io"
	"log/slog"
	"os"
)

// Options holds configuration for the logger.
// Use Option functions to modify.
// Zero value produces an info level non-colored logger to stderr.
// Service is added as attribute to each log line when non-empty.
// Writer defaults to os.Stderr when nil.
// AddSource toggles attachment of source file and line.
// Color toggles ANSI coloring (wraps whole JSON line) relying on terminal support.
// Level defaults to slog.LevelInfo if unset (0) but can be debug (<0) or warn/error (>0).
// NOTE: When coloring, ANSI sequences render the line invalid JSON for machine parsing.
// If you need machine-readable logs, disable color or redirect to a file.
// We still keep embedded JSON inside the color wrapper so humans see colors, machines can strip codes.
// (Many collectors auto-strip ANSI).
// Service appears in attribute key "service".
// Future: Add optional sampling, redaction, etc.

type Options struct {
	Level     slog.Level
	Service   string
	Color     bool
	AddSource bool
	Writer    io.Writer
}

type Option func(*Options)

// WithLevel sets minimum level.
func WithLevel(l slog.Level) Option { return func(o *Options) { o.Level = l } }

// WithService sets a service name attribute.
func WithService(s string) Option { return func(o *Options) { o.Service = s } }

// WithColor toggles ANSI color output.
func WithColor(enabled bool) Option { return func(o *Options) { o.Color = enabled } }

// WithAddSource toggles source file:line.
func WithAddSource(enabled bool) Option { return func(o *Options) { o.AddSource = enabled } }

// WithWriter sets custom destination writer.
func WithWriter(w io.Writer) Option { return func(o *Options) { o.Writer = w } }

func applyOptions(opts []Option) Options {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}
	if o.Writer == nil {
		o.Writer = os.Stderr
	}
	if o.Level == 0 {
		o.Level = slog.LevelInfo
	}
	return o
}
