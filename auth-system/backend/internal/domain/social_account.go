// internal/domain/social_account.go
// Sprint 6 — Google Social Login (US-13).
// SocialAccount links a social provider identity (Google) to a local user.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SocialAccount links a social provider identity to a local user record.
// One row per (provider, provider_user_id) pair — backed by oauth_social_accounts table.
type SocialAccount struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       string     // e.g. "google"
	ProviderUserID string     // Google "sub" claim
	Email          string     // email as returned by the provider
	AccessToken    string     // may be empty if not stored
	RefreshToken   string     // may be empty if not stored
	ExpiresAt      *time.Time // token expiry; nil if not provided by provider
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SocialAccountRepository defines data operations on oauth_social_accounts.
// All methods operate within the current tenant schema (set by middleware).
type SocialAccountRepository interface {
	// FindByProvider returns the social account for a given provider + provider user ID.
	FindByProvider(ctx context.Context, provider, providerUserID string) (*SocialAccount, error)

	// Create inserts a new social account link.
	Create(ctx context.Context, acct *SocialAccount) error

	// UpdateTokens refreshes the stored access/refresh tokens and expiry for an account.
	UpdateTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiresAt *time.Time) error
}

// Social account domain errors.
var (
	ErrSocialAccountNotFound      = errors.New("social account not found")
	ErrSocialAccountAlreadyExists = errors.New("social account already linked")

	// ErrGoogleNotConfigured is returned when the tenant has no Google OAuth credentials
	// and no global fallback credentials are configured.
	ErrGoogleNotConfigured = errors.New("Google OAuth is not configured for this tenant")

	// ErrGoogleStateInvalid is returned when the OAuth state parameter is missing,
	// expired, or does not match the stored value (CSRF protection).
	ErrGoogleStateInvalid = errors.New("invalid or expired OAuth state parameter")

	// ErrPasswordRequiredForLinking is returned (per ADR-006) when a Google callback
	// email matches an existing password-based account and the caller has not provided
	// the current password to authorize the link.
	ErrPasswordRequiredForLinking = errors.New("password verification required before linking Google account")
)
