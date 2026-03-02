// internal/domain/session.go
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session represents a user's authenticated session, anchored by a refresh token.
type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	FamilyID         uuid.UUID
	IPAddress        string
	UserAgent        string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	LastUsedAt       time.Time
	RevokedAt        *time.Time
	IsRevoked        bool
}

// IsValid returns true if the session can be used for token refresh.
func (s *Session) IsValid() bool {
	return !s.IsRevoked && time.Now().Before(s.ExpiresAt)
}

// IsExpired returns true if the session has passed its absolute expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// RefreshTokenValue is a value object representing the raw opaque refresh token.
type RefreshTokenValue struct {
	Raw    string
	Hash   string
	Family uuid.UUID
}

// Session domain errors.
var (
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionRevoked          = errors.New("session has been revoked")
	ErrSessionExpired          = errors.New("session has expired")
	ErrSuspiciousTokenReuse    = errors.New("token reuse detected — all sessions in family revoked")
	ErrInvalidRefreshToken     = errors.New("invalid refresh token")
)
