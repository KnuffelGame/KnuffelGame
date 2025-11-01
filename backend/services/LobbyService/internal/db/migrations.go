package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

// RunMigrations runs all pending database migrations from the specified directory
func RunMigrations(db *sql.DB, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed successfully")
	return nil
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(db *sql.DB, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	return goose.Status(db, migrationsDir)
}
