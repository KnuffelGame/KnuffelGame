package models

import "testing"

func TestCreateJWTRequestValidation_Success(t *testing.T) {
	r := CreateJWTRequest{UserID: "550e8400-e29b-41d4-a716-446655440000", Username: "Alice 123"}
	errMap := r.Validate()
	if len(errMap) != 0 {
		t.Fatalf("expected no errors, got %v", errMap)
	}
}

func TestCreateJWTRequestValidation_FailLength(t *testing.T) {
	r := CreateJWTRequest{UserID: "550e8400-e29b-41d4-a716-446655440000", Username: "Al"}
	errMap := r.Validate()
	if errMap["Username"] != "min" {
		t.Fatalf("expected min error, got %v", errMap)
	}
}

func TestCreateJWTRequestValidation_FailChars(t *testing.T) {
	r := CreateJWTRequest{UserID: "550e8400-e29b-41d4-a716-446655440000", Username: "Alice!"}
	errMap := r.Validate()
	if errMap["Username"] != "usernameFmt" {
		t.Fatalf("expected usernameFmt error, got %v", errMap)
	}
}

func TestCreateJWTRequestValidation_FailEmpty(t *testing.T) {
	r := CreateJWTRequest{Username: ""}
	errMap := r.Validate()
	if errMap["Username"] != "required" {
		t.Fatalf("expected required error, got %v", errMap)
	}
}

func TestValidateJWTRequestValidation_Success(t *testing.T) {
	r := ValidateJWTRequest{Token: "header.payload.signature"}
	errMap := r.Validate()
	if len(errMap) != 0 {
		t.Fatalf("expected no errors, got %v", errMap)
	}
}

func TestValidateJWTRequestValidation_Fail(t *testing.T) {
	r := ValidateJWTRequest{Token: "invalid"}
	errMap := r.Validate()
	if errMap["Token"] != "jwt" {
		t.Fatalf("expected jwt error, got %v", errMap)
	}
}
