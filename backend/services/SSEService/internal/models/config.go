package models

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration for the SSEService.
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

	// Rate limiting (requests per minute per IP)
	RateLimitInternalPerMinute int
	RateLimitSSEPerMinute      int

	// Auth / tokens
	JWTCookieName string
	InternalToken string // env SSE_INTERNAL_TOKEN; required for /internal

	// SSE behavior
	HeartbeatInterval time.Duration // default 30s
	APIGatewayBaseURL string        // for authorizer calls (future use)
}

// Load reads configuration values from environment variables with sensible defaults.
// List values are comma-separated (CSV style).
func Load() *Config {
	cfg := &Config{
		Port:        getenvDefault("PORT", "8084"),
		ServiceName: getenvDefault("SERVICE_NAME", "SSEService"),
		LogLevel:    getenvDefault("LOG_LEVEL", "info"),
		LogColor:    parseBoolEnv("LOG_COLOR", false),

		CORSAllowOrigins:     parseCSVEnv("CORS_ALLOW_ORIGINS", []string{"http://localhost:5173", "http://localhost:8080"}),
		CORSAllowMethods:     parseCSVEnv("CORS_ALLOW_METHODS", []string{"GET", "POST", "OPTIONS"}),
		CORSAllowHeaders:     parseCSVEnv("CORS_ALLOW_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"}),
		CORSAllowCredentials: parseBoolEnv("CORS_ALLOW_CREDENTIALS", true),

		RateLimitInternalPerMinute: parseIntEnv("RATE_LIMIT_INTERNAL_PER_MINUTE", 60),
		RateLimitSSEPerMinute:      parseIntEnv("RATE_LIMIT_SSE_PER_MINUTE", 0), // 0 = disabled

		JWTCookieName: getenvDefault("JWT_COOKIE_NAME", "jwt"),
		InternalToken: os.Getenv("SSE_INTERNAL_TOKEN"),

		HeartbeatInterval: parseHeartbeatEnv("HEARTBEAT_INTERVAL_MS", 30_000), // ms
		APIGatewayBaseURL: os.Getenv("APIGATEWAY_BASE_URL"),
	}
	return cfg
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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
	case "":
		return def
	default:
		return def
	}
}

func parseIntEnv(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func parseHeartbeatEnv(key string, defMs int) time.Duration {
	ms := parseIntEnv(key, defMs)
	if ms <= 0 {
		ms = defMs
	}
	return time.Duration(ms) * time.Millisecond
}
