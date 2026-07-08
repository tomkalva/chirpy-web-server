package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validToken, _ := MakeJWT(userID, "secret", time.Hour)

	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantUserID  uuid.UUID
		wantErr     bool
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			tokenSecret: "secret",
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "Invalid token",
			tokenString: "invalid.token.string",
			tokenSecret: "secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "Wrong secret",
			tokenString: validToken,
			tokenSecret: "wrong_secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		input     http.Header
		expected  string
		expectErr bool
	}{
		{
			name: "real token",
			input: http.Header{
				"Authorization": []string{"Bearer 123456789"},
			},
			expected:  "123456789",
			expectErr: false,
		},
		{
			name: "missing header",
			input: http.Header{
				"Something": []string{"here"},
			},
			expected:  "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetBearerToken(tc.input)
			if result != tc.expected {
				t.Errorf("GetBearerToken(), case: %v, got: %v, expected: %v", tc.name, result, tc.expected)
			}
			if (err != nil) != tc.expectErr {
				t.Errorf("GetBearerToken() error = %v, expectErr %v", err, tc.expectErr)
			}
		})
	}
}
