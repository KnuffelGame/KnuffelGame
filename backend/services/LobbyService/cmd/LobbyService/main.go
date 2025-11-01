package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	router "github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/internal/db"
	"github.com/KnuffelGame/KnuffelGame/backend/services/LobbyService/pkg/config"
)

func main() {
	// ensure SERVICE_NAME env is present (fallback if empty)
	if os.Getenv("SERVICE_NAME") == "" {
		_ = os.Setenv("SERVICE_NAME", "LobbyService")
	}
	log := logger.FromEnv().With(slog.String("component", "bootstrap"))

	cfg := config.Load()

	// Initialize database connection
	dbConfig := db.Config{
		Host:     cfg.DatabaseHost,
		Port:     cfg.DatabasePort,
		User:     cfg.DatabaseUser,
		Password: cfg.DatabasePassword,
		Database: cfg.DatabaseName,
		SSLMode:  cfg.DatabaseSSLMode,
	}

	dbConn, err := db.New(dbConfig)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbConn.Close()

	// Run database migrations
	if err := db.RunMigrations(dbConn.DB); err != nil {
		log.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	r := router.New()
	log.Info("listening", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Error("server exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
