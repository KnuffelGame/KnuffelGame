package config

import "os"

// Config holds runtime configuration loaded from environment variables.
// PORT defaults to 8084 if unset.
// JWT_SECRET must be set for authentication (warning logged otherwise).
// Extend here for future configuration values.

type Config struct {
	JWTSecret string
	Port      string
}

func Load() *Config {
	secret := os.Getenv("JWT_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084" // SSE Service default port from docker-compose.yaml
	}
	return &Config{JWTSecret: secret, Port: port}
}
