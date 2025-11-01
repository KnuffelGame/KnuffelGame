package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	router "github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/pkg/config"
)

func main() {
	// ensure SERVICE_NAME env is present (fallback if empty)
	if os.Getenv("SERVICE_NAME") == "" {
		_ = os.Setenv("SERVICE_NAME", "AuthService")
	}
	log := logger.FromEnv().With(slog.String("component", "bootstrap"))

	cfg := config.Load()

	r := router.New()
	log.Info("listening", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Error("server exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
