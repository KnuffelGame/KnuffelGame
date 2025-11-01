package config

import "os"

// Config holds runtime configuration loaded from environment variables.
// PORT defaults to 8081 if unset.
// JWT_SECRET must be set for token generation (warning logged otherwise).
// Extend here for future configuration values.

type Config struct {
	Port string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	return &Config{Port: port}
}
