package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	router "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

func main() {
	// Load configuration from environment with sensible defaults
	cfg := models.Load()

	// Ensure SERVICE_NAME env is present (so logger.FromEnv users also benefit)
	if os.Getenv("SERVICE_NAME") == "" {
		_ = os.Setenv("SERVICE_NAME", cfg.ServiceName)
	}

	// Initialize logger bound to config options
	level := parseLevel(cfg.LogLevel)
	log := logger.New(
		logger.WithService(cfg.ServiceName),
		logger.WithLevel(level),
		logger.WithColor(cfg.LogColor),
	).With(slog.String("component", "bootstrap"))

	// Warn if internal token is missing; internal endpoints will be blocked
	if cfg.InternalToken == "" {
		log.Warn("SSE_INTERNAL_TOKEN is empty; internal endpoints will be blocked")
	}

	// Build router with middleware and route stubs
	r := router.New(cfg, log)

	// HTTP server with timeouts
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server
	go func() {
		log.Info("listening", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server exited", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
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

// parseLevel maps string level to slog.Level with info as default.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
