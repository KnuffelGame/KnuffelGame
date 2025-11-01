package joincode

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	// CodeLength is the length of the join code
	CodeLength = 6
	// Charset contains all valid characters for the join code
	Charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// MaxRetries is the maximum number of attempts to generate a unique code
	MaxRetries = 5
)

var (
	// ErrMaxRetriesExceeded is returned when we can't generate a unique code after MaxRetries attempts
	ErrMaxRetriesExceeded = errors.New("failed to generate unique join code after maximum retries")
	// ErrInvalidDB is returned when the database connection is nil
	ErrInvalidDB = errors.New("database connection is nil")
)

// Generator handles the generation of unique join codes
type Generator struct {
	db *sql.DB
}

// NewGenerator creates a new join code generator
func NewGenerator(db *sql.DB) *Generator {
	return &Generator{db: db}
}

// GenerateJoinCode generates a unique 6-character alphanumeric join code
// Format: 6 characters, A-Z + 0-9 (e.g. "ABC123")
// Returns uppercase code and error if unable to generate unique code
func (g *Generator) GenerateJoinCode() (string, error) {
	if g.db == nil {
		return "", ErrInvalidDB
	}

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		code, err := generateRandomCode()
		if err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}

		// Normalize to uppercase
		code = strings.ToUpper(code)

		// Check for collision in database
		exists, err := g.codeExists(code)
		if err != nil {
			return "", fmt.Errorf("failed to check code collision: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", ErrMaxRetriesExceeded
}

// generateRandomCode generates a random code using crypto/rand
func generateRandomCode() (string, error) {
	bytes := make([]byte, CodeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	charsetLen := len(Charset)
	for i := 0; i < CodeLength; i++ {
		bytes[i] = Charset[int(bytes[i])%charsetLen]
	}

	return string(bytes), nil
}

// codeExists checks if a join code already exists in the database
func (g *Generator) codeExists(code string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM lobbies WHERE join_code = $1)`
	err := g.db.QueryRow(query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to query join code: %w", err)
	}
	return exists, nil
}
