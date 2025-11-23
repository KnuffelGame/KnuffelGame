package db

import (
	"database/sql"
	"fmt"

	// WICHTIG: Der Treiber-Import (Adapter für pgx)
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// New erstellt eine neue Datenbankverbindung via database/sql
// Wir geben direkt *sql.DB zurück, da das der Standard für Go ist.
func New(cfg Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	// Öffnet die Verbindung (lazy)
	// "pgx" ist der Treiber-Name, den wir oben importiert haben
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	// Ping prüft, ob die Verbindung wirklich funktioniert (Fail Fast)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
	}

	return db, nil
}
