package config

import "os"

// Config holds runtime configuration loaded from environment variables.
// PORT defaults to 8083 if unset.
// Database configuration must be provided via environment variables.
// Extend here for future configuration values.

type Config struct {
	Port             string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	dbHost := os.Getenv("DATABASE_HOST")
	if dbHost == "" {
		dbHost = "Postgres"
	}

	dbPort := os.Getenv("DATABASE_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("DATABASE_USER")
	if dbUser == "" {
		dbUser = "lobby"
	}

	dbPassword := os.Getenv("DATABASE_PASSWORD")
	if dbPassword == "" {
		dbPassword = "secure"
	}

	dbName := os.Getenv("DATABASE_NAME")
	if dbName == "" {
		dbName = "lobby"
	}

	dbSSLMode := os.Getenv("DATABASE_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	return &Config{
		Port:             port,
		DatabaseHost:     dbHost,
		DatabasePort:     dbPort,
		DatabaseUser:     dbUser,
		DatabasePassword: dbPassword,
		DatabaseName:     dbName,
		DatabaseSSLMode:  dbSSLMode,
	}
}
