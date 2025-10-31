package logger

import (
	"context"
	"log/slog"
)

// ctxKey prevents collisions.
var ctxKey struct{}

// WithLogger stores logger in context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, l)
}

// Logger retrieves logger from context or returns Default().
func Logger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}
	if l, ok := ctx.Value(ctxKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return Default()
}
