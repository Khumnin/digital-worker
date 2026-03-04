// internal/domain/mfa.go
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// MFABackupCode is a single-use recovery code generated at TOTP enrollment.
// The plaintext code is only returned once at enrollment; only the SHA-256
// hash is persisted.
type MFABackupCode struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CodeHash  string // SHA-256 hex hash; plaintext is never stored
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

// MFARepository defines data operations for MFA backup codes.
// All methods operate within the tenant schema identified by context.
type MFARepository interface {
	// CreateBackupCodes stores hashed backup codes for a user, replacing any
	// existing unused codes atomically.
	CreateBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error

	// ConsumeBackupCode finds a matching unused backup code by its hash and
	// marks it used in a single atomic operation.
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string) error

	// DeleteAllForUser removes all backup codes for the user. Called when MFA
	// is disabled.
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}

// MFA-specific sentinel errors.
var (
	ErrMFAAlreadyEnabled      = errors.New("MFA is already enabled for this account")
	ErrMFANotEnabled          = errors.New("MFA is not enabled for this account")
	ErrInvalidTOTPCode        = errors.New("TOTP code is invalid or has expired")
	ErrMFARequired            = errors.New("MFA verification required")
	ErrMFAEnrollmentRequired  = errors.New("MFA enrollment is required for this organization")
	ErrBackupCodeInvalid      = errors.New("backup code is invalid or already used")
	ErrTOTPRateLimited        = errors.New("too many TOTP verification attempts")
)
