package joincode

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestGenerateRandomCode tests the random code generation format
func TestGenerateRandomCode(t *testing.T) {
	// Test multiple generations to ensure consistency
	for i := 0; i < 100; i++ {
		code, err := generateRandomCode()
		if err != nil {
			t.Fatalf("generateRandomCode() failed: %v", err)
		}

		// Check length
		if len(code) != CodeLength {
			t.Errorf("Expected code length %d, got %d", CodeLength, len(code))
		}

		// Check format: only A-Z and 0-9
		matched, err := regexp.MatchString("^[A-Z0-9]+$", code)
		if err != nil {
			t.Fatalf("regex error: %v", err)
		}
		if !matched {
			t.Errorf("Code %s does not match expected format [A-Z0-9]", code)
		}
	}
}

// TestGenerateRandomCodeUniqueness tests that generated codes are diverse
func TestGenerateRandomCodeUniqueness(t *testing.T) {
	// Generate multiple codes and check for diversity
	codes := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		code, err := generateRandomCode()
		if err != nil {
			t.Fatalf("generateRandomCode() failed: %v", err)
		}
		codes[code] = true
	}

	// We expect high diversity - at least 90% unique codes
	minUniqueExpected := int(float64(iterations) * 0.9)
	if len(codes) < minUniqueExpected {
		t.Errorf("Expected at least %d unique codes, got %d", minUniqueExpected, len(codes))
	}
}

// TestGenerateJoinCode_Success tests successful code generation
func TestGenerateJoinCode_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)

	// Mock database query to return false (code doesn't exist)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	code, err := generator.GenerateJoinCode()
	if err != nil {
		t.Fatalf("GenerateJoinCode() failed: %v", err)
	}

	// Verify code format
	if len(code) != CodeLength {
		t.Errorf("Expected code length %d, got %d", CodeLength, len(code))
	}

	// Verify uppercase
	matched, _ := regexp.MatchString("^[A-Z0-9]+$", code)
	if !matched {
		t.Errorf("Code %s is not uppercase alphanumeric", code)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestGenerateJoinCode_Collision tests collision handling with retry
func TestGenerateJoinCode_Collision(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)

	// First two attempts have collisions, third succeeds
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	code, err := generator.GenerateJoinCode()
	if err != nil {
		t.Fatalf("GenerateJoinCode() failed: %v", err)
	}

	if len(code) != CodeLength {
		t.Errorf("Expected code length %d, got %d", CodeLength, len(code))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestGenerateJoinCode_MaxRetriesExceeded tests behavior when max retries exceeded
func TestGenerateJoinCode_MaxRetriesExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)

	// All attempts result in collision
	for i := 0; i < MaxRetries; i++ {
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	}

	code, err := generator.GenerateJoinCode()
	if err == nil {
		t.Fatal("Expected error when max retries exceeded, got nil")
	}

	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Expected ErrMaxRetriesExceeded, got %v", err)
	}

	if code != "" {
		t.Errorf("Expected empty code on error, got %s", code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestGenerateJoinCode_NilDB tests handling of nil database
func TestGenerateJoinCode_NilDB(t *testing.T) {
	generator := NewGenerator(nil)

	code, err := generator.GenerateJoinCode()
	if err == nil {
		t.Fatal("Expected error with nil DB, got nil")
	}

	if !errors.Is(err, ErrInvalidDB) {
		t.Errorf("Expected ErrInvalidDB, got %v", err)
	}

	if code != "" {
		t.Errorf("Expected empty code on error, got %s", code)
	}
}

// TestGenerateJoinCode_DatabaseError tests handling of database errors
func TestGenerateJoinCode_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)

	// Mock database error
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	code, err := generator.GenerateJoinCode()
	if err == nil {
		t.Fatal("Expected error when database query fails, got nil")
	}

	if code != "" {
		t.Errorf("Expected empty code on error, got %s", code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestCodeFormat_AlwaysUppercase tests that generated codes are always uppercase
func TestCodeFormat_AlwaysUppercase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)

	// Generate multiple codes and verify all are uppercase
	for i := 0; i < 10; i++ {
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lobbies WHERE join_code = \$1\)`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		code, err := generator.GenerateJoinCode()
		if err != nil {
			t.Fatalf("GenerateJoinCode() failed: %v", err)
		}

		// Check that code equals its uppercase version
		if code != code {
			t.Errorf("Code %s is not uppercase", code)
		}

		// Check no lowercase letters
		matched, _ := regexp.MatchString("[a-z]", code)
		if matched {
			t.Errorf("Code %s contains lowercase letters", code)
		}
	}
}

// TestNewGenerator tests the constructor
func TestNewGenerator(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()

	generator := NewGenerator(db)
	if generator == nil {
		t.Fatal("Expected non-nil generator")
	}

	if generator.db != db {
		t.Error("Generator database not set correctly")
	}
}
