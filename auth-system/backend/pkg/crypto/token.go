// pkg/crypto/token.go
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const refreshTokenBytes = 32

// GenerateOpaqueToken generates a cryptographically random opaque token.
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken computes the SHA-256 hash of a token for database storage.
func HashToken(raw string) (string, string) {
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:])
}

// GenerateTokenWithHash generates a new opaque token and returns both the
// raw value (for the client) and the SHA-256 hash (for the database).
func GenerateTokenWithHash() (raw string, hash string, err error) {
	raw, err = GenerateOpaqueToken()
	if err != nil {
		return "", "", err
	}
	_, hash = HashToken(raw)
	return raw, hash, nil
}

// HashTokenString returns just the SHA-256 hex hash of the given raw token string.
func HashTokenString(raw string) string {
	_, hash := HashToken(raw)
	return hash
}
