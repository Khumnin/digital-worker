// pkg/crypto/crypto_test.go
package crypto_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"tigersoft/auth-system/pkg/crypto"
)

// ---------------------------------------------------------------------------
// HashPassword / VerifyPassword
// ---------------------------------------------------------------------------

func TestHashPassword_ProducesArgon2idFormat(t *testing.T) {
	hash, err := crypto.HashPassword("CorrectHorse99!")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected hash to start with $argon2id$, got %q", hash)
	}
}

func TestHashPassword_ProducesDifferentHashesForSamePassword(t *testing.T) {
	pw := "CorrectHorse99!"
	h1, err1 := crypto.HashPassword(pw)
	h2, err2 := crypto.HashPassword(pw)
	if err1 != nil || err2 != nil {
		t.Fatalf("HashPassword errors: %v / %v", err1, err2)
	}
	if h1 == h2 {
		t.Error("expected two hashes of the same password to differ (random salt)")
	}
}

func TestVerifyPassword_CorrectPassword(t *testing.T) {
	pw := "MyS3cur3P@ss!"
	hash, err := crypto.HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !crypto.VerifyPassword(pw, hash) {
		t.Error("expected VerifyPassword to return true for correct password")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := crypto.HashPassword("CorrectPassword1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if crypto.VerifyPassword("WrongPassword1!", hash) {
		t.Error("expected VerifyPassword to return false for wrong password")
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, err := crypto.HashPassword("NonEmptyPassword1!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if crypto.VerifyPassword("", hash) {
		t.Error("expected VerifyPassword to return false for empty password")
	}
}

func TestVerifyPassword_InvalidHashFormat(t *testing.T) {
	if crypto.VerifyPassword("anything", "not-a-valid-hash") {
		t.Error("expected VerifyPassword to return false for malformed hash")
	}
}

func TestHashPassword_EmptyPassword_Hashes(t *testing.T) {
	// Hashing an empty string should still succeed (policy enforcement is
	// a separate layer); the argon2id function accepts any byte slice.
	hash, err := crypto.HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword(\"\") returned error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected argon2id prefix, got %q", hash)
	}
	// Empty password should verify against its own hash.
	if !crypto.VerifyPassword("", hash) {
		t.Error("expected VerifyPassword to return true for empty password against its own hash")
	}
}

// ---------------------------------------------------------------------------
// GenerateTokenWithHash
// ---------------------------------------------------------------------------

func TestGenerateTokenWithHash_NonEmpty(t *testing.T) {
	raw, hash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		t.Fatalf("GenerateTokenWithHash: %v", err)
	}
	if raw == "" {
		t.Error("raw token must not be empty")
	}
	if hash == "" {
		t.Error("hash must not be empty")
	}
}

func TestGenerateTokenWithHash_HashDiffersFromRaw(t *testing.T) {
	raw, hash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		t.Fatalf("GenerateTokenWithHash: %v", err)
	}
	if raw == hash {
		t.Error("hash must differ from raw token")
	}
}

func TestGenerateTokenWithHash_TwoCallsProduceDifferentTokens(t *testing.T) {
	raw1, hash1, _ := crypto.GenerateTokenWithHash()
	raw2, hash2, _ := crypto.GenerateTokenWithHash()
	if raw1 == raw2 {
		t.Error("successive raw tokens must differ")
	}
	if hash1 == hash2 {
		t.Error("successive hashes must differ")
	}
}

// ---------------------------------------------------------------------------
// HashTokenString
// ---------------------------------------------------------------------------

func TestHashTokenString_Is64HexChars(t *testing.T) {
	input := "some-opaque-token-value"
	h := crypto.HashTokenString(input)
	if len(h) != 64 {
		t.Errorf("expected 64 hex chars (SHA-256), got %d chars: %q", len(h), h)
	}
	// Verify it is valid hex.
	if _, err := hex.DecodeString(h); err != nil {
		t.Errorf("HashTokenString returned non-hex string: %v", err)
	}
}

func TestHashTokenString_IsDeterministic(t *testing.T) {
	input := "deterministic-token"
	h1 := crypto.HashTokenString(input)
	h2 := crypto.HashTokenString(input)
	if h1 != h2 {
		t.Errorf("expected same hash for same input: %q vs %q", h1, h2)
	}
}

func TestHashTokenString_DifferentInputsDifferentHashes(t *testing.T) {
	h1 := crypto.HashTokenString("token-a")
	h2 := crypto.HashTokenString("token-b")
	if h1 == h2 {
		t.Error("different inputs must produce different SHA-256 hashes")
	}
}

func TestHashTokenString_EmptyInput(t *testing.T) {
	h := crypto.HashTokenString("")
	// SHA-256("") is known: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	const sha256Empty = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if h != sha256Empty {
		t.Errorf("expected SHA-256 of empty string %q, got %q", sha256Empty, h)
	}
}
