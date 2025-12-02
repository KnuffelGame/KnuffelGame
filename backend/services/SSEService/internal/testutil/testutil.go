// Package testutil provides shared testing utilities for SSEService.
package testutil

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/KnuffelGame/KnuffelGame/backend/libs/logger"
	router "github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal"
	"github.com/KnuffelGame/KnuffelGame/backend/services/SSEService/internal/models"
)

// NewTestConfig returns a minimal config for tests with short heartbeat and jwt cookie name "jwt".
func NewTestConfig() *models.Config {
	return &models.Config{
		Port:        "0",
		ServiceName: "SSEServiceTest",
		LogLevel:    "error",
		LogColor:    false,

		CORSAllowOrigins:     []string{"http://localhost"},
		CORSAllowMethods:     []string{"GET", "POST", "OPTIONS"},
		CORSAllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-Requested-With", "Cookie", "X-Request-ID"},
		CORSAllowCredentials: true,

		RateLimitInternalPerMinute: 0,
		RateLimitSSEPerMinute:      0,

		JWTCookieName: "jwt",
		InternalToken: "test-internal",

		HeartbeatInterval: 100 * time.Millisecond,
		APIGatewayBaseURL: "",
	}
}

// NewTestLogger creates a quiet slog.Logger suitable for tests.
func NewTestLogger() *slog.Logger {
	return logger.New(
		logger.WithLevel(slog.LevelError),
		logger.WithColor(false),
		logger.WithWriter(io.Discard),
		logger.WithService("SSEServiceTest"),
	)
}

// NewTestRouter builds a chi router using the provided config and logger.
// When mockAPIGatewayBaseURL is non-empty, it is injected into cfg.APIGatewayBaseURL.
func NewTestRouter(cfg *models.Config, log *slog.Logger, mockAPIGatewayBaseURL string) http.Handler {
	if cfg == nil {
		cfg = NewTestConfig()
	}
	if log == nil {
		log = NewTestLogger()
	}
	if mockAPIGatewayBaseURL != "" {
		cfg.APIGatewayBaseURL = mockAPIGatewayBaseURL
	}
	return router.New(cfg, log)
}
