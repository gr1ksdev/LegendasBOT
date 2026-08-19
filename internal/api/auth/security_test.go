package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// issuer is used in JWT token creation for testing
var issuer = "freddybot"

func TestJWTSecurity(t *testing.T) {
	// Setup a valid secret for testing
	originalKey := secreteKey
	secreteKey = []byte("this-is-a-very-secret-key-with-more-than-32-chars")
	defer func() { secreteKey = originalKey }()

	t.Run("None Algorithm Attack", func(t *testing.T) {
		// Create a token with "none" algorithm
		token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
			UserID: 123,
			Role:   RoleUser,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Issuer:    issuer,
			},
		})

		// SigningMethodNone returns an empty string and no error with jwt.UnsafeAllowNoneSignatureType
		tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

		_, err := ValidateToken(tokenStr)
		if err == nil {
			t.Errorf("expected error for 'none' algorithm token, got nil")
		} else {
			expectedErr := "unexpected signing method: none"
			if !strings.Contains(err.Error(), expectedErr) {
				t.Errorf("expected error to contain '%s', got '%v'", expectedErr, err)
			}
		}
	})

	t.Run("Wrong Signing Method (RSA confusion simulation)", func(t *testing.T) {
		// Even if we don't have an RSA key here, we can simulate a token with a different alg
		token := jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{
			UserID: 123,
			Role:   RoleUser,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Issuer:    issuer,
			},
		})

		tokenStr, _ := token.SignedString(secreteKey)

		_, err := ValidateToken(tokenStr)
		if err == nil {
			t.Errorf("expected error for HS384 algorithm token, got nil")
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		tokenStr, _ := GenerateToken(123, RoleUser, 1, -time.Hour) // Already expired

		_, err := ValidateToken(tokenStr)
		if err == nil {
			t.Errorf("expected error for expired token, got nil")
		}
	})

	t.Run("Invalid Signature (Tampering)", func(t *testing.T) {
		tokenStr, _ := GenerateToken(123, RoleUser, 1, time.Hour)

		// Tamper with the token (modify one character in the payload part)
		// JWT is header.payload.signature
		_, _, _ = jwt.NewParser().ParseUnverified(tokenStr, &Claims{})

		tamperedTokenStr := tokenStr[:len(tokenStr)-10] + "tampered" // just break the signature

		_, err := ValidateToken(tamperedTokenStr)
		if err == nil {
			t.Errorf("expected error for tampered token signature, got nil")
		}
	})

	t.Run("Information Leakage - Header", func(t *testing.T) {
		tokenStr, _ := GenerateToken(123, RoleUser, 1, time.Hour)
		token, _, _ := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})

		// Check if we are leaking sensitive info in the header
		if _, ok := token.Header["secret"]; ok {
			t.Errorf("sensitive info leaked in JWT header")
		}
	})
}

func TestSecretKeyRequirement(t *testing.T) {
	// Temporarily set a short key
	originalKey := secreteKey
	secreteKey = []byte("short")
	defer func() { secreteKey = originalKey }()

	_, err := GenerateToken(123, RoleUser, 1, time.Hour)
	if err == nil {
		t.Errorf("expected error for short secret key, got nil")
	} else if err.Error() != "JWT secret too short" {
		t.Errorf("unexpected error message: %v", err)
	}
}
