package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/leirbagxis/FreddyBot/internal/telegram/mtproto/encryption"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		plaintext []byte
	}{
		{
			name:      "short key with text data",
			key:       "my-secret-key-123",
			plaintext: []byte("Hello, World! This is a test message."),
		},
		{
			name:      "long hex key with binary data",
			key:       "75d1ffcbc272c0829d35cf7dd59e55f2b5bb78a24e13fdbae08cbdf88863a04e2f51f79051286e09a7f9c0aca070e7e9a66287d572982dc951659ca9b8ee9dec",
			plaintext: []byte("{\"session_key\":\"abc123\",\"dc\":2,\"auth_key\":\"some_auth_key_data_here\"}"),
		},
		{
			name:      "empty plaintext",
			key:       "test-key",
			plaintext: []byte{},
		},
		{
			name:      "single byte",
			key:       "a",
			plaintext: []byte{0x42},
		},
		{
			name:      "large data (10KB)",
			key:       "large-test-key",
			plaintext: randomBytes(t, 10240),
		},
		{
			name:      "session-like data with null bytes",
			key:       "session-encryption-key-2024",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x7F, 0x80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := encryption.DeriveKey(tt.key)

			// Encrypt
			cipherHex, err := encryption.Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if cipherHex == "" && len(tt.plaintext) > 0 {
				t.Fatal("Encrypt() returned empty string for non-empty plaintext")
			}

			// Decrypt
			decrypted, err := encryption.Decrypt(cipherHex, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Compare
			if len(tt.plaintext) != len(decrypted) {
				t.Fatalf("Decrypt() length mismatch: got %d, want %d", len(decrypted), len(tt.plaintext))
			}

			for i := range tt.plaintext {
				if tt.plaintext[i] != decrypted[i] {
					t.Fatalf("Decrypt() byte mismatch at position %d: got %02x, want %02x", i, decrypted[i], tt.plaintext[i])
				}
			}
		})
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := encryption.DeriveKey("test-key")
	plaintext := []byte("sensitive session data")

	cipherHex, err := encryption.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with the hex string
	tampered := string(cipherHex)
	if len(tampered) > 10 {
		// Change a character in the middle
		bytes := []byte(tampered)
		bytes[len(bytes)/2] ^= 0xFF
		tampered = string(bytes)
	}

	_, err = encryption.Decrypt(tampered, key)
	if err == nil {
		t.Error("Decrypt() should error on tampered ciphertext, got nil")
	}
}

func TestDecrypt_InvalidHex(t *testing.T) {
	key := encryption.DeriveKey("test-key")

	invalidInputs := []string{
		"",
		"not-hex-string!!!",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		"01",
		"000", // odd length
	}

	for _, input := range invalidInputs {
		t.Run("input_"+input[:min(len(input), 8)], func(t *testing.T) {
			_, err := encryption.Decrypt(input, key)
			if err == nil {
				t.Errorf("Decrypt(%q) should error, got nil", input)
			}
		})
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := encryption.DeriveKey("correct-key")
	key2 := encryption.DeriveKey("wrong-key")

	plaintext := []byte("important session data")
	cipherHex, err := encryption.Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = encryption.Decrypt(cipherHex, key2)
	if err == nil {
		t.Error("Decrypt() with wrong key should error, got nil")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := encryption.DeriveKey("test-key")

	// GCM nonce is typically 12 bytes, so anything shorter should fail
	shortHex := "aabbccdd" // 4 bytes
	_, err := encryption.Decrypt(shortHex, key)
	if err == nil {
		t.Error("Decrypt() with too short ciphertext should error, got nil")
	}
}

func TestDeriveKey_Consistency(t *testing.T) {
	key1 := encryption.DeriveKey("same-key")
	key2 := encryption.DeriveKey("same-key")
	key3 := encryption.DeriveKey("different-key")

	if len(key1) != 32 {
		t.Fatalf("DeriveKey() returned key of length %d, want 32", len(key1))
	}

	// Same input = same output
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("DeriveKey() is not deterministic for same input")
		}
	}

	// Different input = different output
	same := true
	for i := range key1 {
		if key1[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("DeriveKey() produced same output for different inputs")
	}
}

func TestEncryptConcurrency(t *testing.T) {
	key := encryption.DeriveKey("concurrency-test-key")
	plaintext := []byte("concurrent encryption test data")
	iterations := 50

	errors := make(chan error, iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			cipherHex, err := encryption.Encrypt(plaintext, key)
			if err != nil {
				errors <- err
				return
			}
			_, err = encryption.Decrypt(cipherHex, key)
			errors <- err
		}()
	}

	for i := 0; i < iterations; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Concurrent encrypt/decrypt error: %v", err)
		}
	}
}

func randomBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatalf("failed to generate random bytes: %v", err)
	}
	return b
}
