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
	router "github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/internal/router"
	"github.com/KnuffelGame/KnuffelGame/backend/services/APIGateway/pkg/config"
)

func main() {
	// 1. Load configuration from environment with sensible defaults
	cfg := config.Load()

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

	// 4. Validate critical configuration (none for now in Phase 3)

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
