// internal/domain/user.go
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusUnverified UserStatus = "unverified"
	UserStatusActive     UserStatus = "active"
	UserStatusDisabled   UserStatus = "disabled"
	UserStatusDeleted    UserStatus = "deleted"
)

// User is the core identity entity. It lives entirely within a tenant schema
// (ADR-001, ADR-002) — there is no global user table.
type User struct {
	ID                uuid.UUID
	Email             string
	PasswordHash      string // Argon2id hash; empty for social-only accounts
	Status            UserStatus
	FirstName         string
	LastName          string
	MFAEnabled        bool
	MFATOTPSecret     *string // Encrypted at rest; nil if MFA not configured
	FailedLoginCount  int
	LockedUntil       *time.Time
	EmailVerifiedAt   *time.Time
	LastLoginAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// IsActive returns true if the user may authenticate.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// IsVerified returns true if the user has confirmed their email address.
func (u *User) IsVerified() bool {
	return u.EmailVerifiedAt != nil
}

// IsLocked returns true if the account is in a temporary lockout period.
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// HasPassword returns true if the user has a local password set.
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}

// FullName returns the display name of the user.
func (u *User) FullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Errors returned from domain validation.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already exists in this tenant")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrAccountDisabled     = errors.New("account disabled")
	ErrAccountLocked       = errors.New("account is temporarily locked")
	ErrAccountDeleted      = errors.New("account has been deleted")
	ErrWeakPassword        = errors.New("password does not meet complexity requirements")
	ErrInvalidEmail        = errors.New("invalid email address format")
)

// emailRegexp is a basic RFC 5322-compatible email validator.
var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NormalizeEmail lowercases and trims whitespace from an email address.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if !emailRegexp.MatchString(normalized) {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// CreateUserInput carries validated input for registering a new user.
type CreateUserInput struct {
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Status       UserStatus
}

// UpdateUserInput carries fields that may be updated on a user record.
type UpdateUserInput struct {
	FirstName        *string
	LastName         *string
	PasswordHash     *string
	Status           *UserStatus
	MFAEnabled       *bool
	MFATOTPSecret    *string
	FailedLoginCount *int
	LockedUntil      *time.Time
	EmailVerifiedAt  *time.Time
	LastLoginAt      *time.Time
	DeletedAt        *time.Time
}
