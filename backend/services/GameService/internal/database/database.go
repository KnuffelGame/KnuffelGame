package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/game_db?sslmode=disable"
	}

	// pgxpool.New erstellt einen neuen Connection Pool.
	// Das ist effizienter als einzelne Verbindungen.
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	// Verbindung mit einem Ping testen.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Pool schließen, wenn Ping fehlschlägt
		return nil, fmt.Errorf("konnte Postgres nicht anpingen: %w", err)
	}

	return pool, nil
}
