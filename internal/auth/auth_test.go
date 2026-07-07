package auth

import (
	"testing"
)

func TestPasswordHashAndCompare(t *testing.T) {
	password := "Sugarfree-winterfresh"

	// Create hash
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("CreateHash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("expected hash, got empty string")
	}

	// Check correct password
	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}

	if !match {
		t.Fatal("expected password to match")
	}

	// Check incorrect password
	match, err = CheckPasswordHash("wrong-password", hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}

	if match {
		t.Fatal("expected wrong password not to match")
	}
}
