// internal/domain/oauth.go
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── OAuth Client ──────────────────────────────────────────────────────────────

// OAuthClient represents a registered OAuth 2.0 client application per tenant.
type OAuthClient struct {
	ID             uuid.UUID
	ClientID       string
	ClientSecret   string // stored as SHA-256 hash; plaintext only returned at registration time
	Name           string
	RedirectURIs   []string
	Scopes         []string
	GrantTypes     []string
	IsConfidential bool
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HasRedirectURI returns true if the given URI is an exact match in the allowed list.
func (c *OAuthClient) HasRedirectURI(uri string) bool {
	for _, u := range c.RedirectURIs {
		if u == uri {
			return true
		}
	}
	return false
}

// HasScope returns true if the given scope is in the client's allowed scope list.
func (c *OAuthClient) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// ── Authorization Code ────────────────────────────────────────────────────────

// AuthorizationCode represents a single-use PKCE authorization code.
type AuthorizationCode struct {
	ID                  uuid.UUID
	CodeHash            string // SHA-256 hash of the raw code; raw code is never stored
	ClientID            string
	UserID              uuid.UUID
	RedirectURI         string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string // always "S256"
	ExpiresAt           time.Time
	Used                bool
	CreatedAt           time.Time
}

// IsExpired returns true if the code's TTL has elapsed.
func (a *AuthorizationCode) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

// ── Domain errors ─────────────────────────────────────────────────────────────

var (
	ErrOAuthClientNotFound      = errors.New("oauth client not found")
	ErrOAuthClientAlreadyExists = errors.New("oauth client already exists")
	ErrInvalidRedirectURI       = errors.New("redirect_uri is not registered for this client")
	ErrInvalidScope             = errors.New("one or more requested scopes are not allowed for this client")
	ErrInvalidCodeChallenge     = errors.New("code_challenge is required and must use S256 method")
	ErrAuthCodeNotFound         = errors.New("authorization code not found or already used")
	ErrAuthCodeExpired          = errors.New("authorization code has expired")
	ErrAuthCodeAlreadyUsed      = errors.New("authorization code has already been used")
	ErrPKCEVerificationFailed   = errors.New("code_verifier does not match code_challenge")
	ErrUnsupportedGrantType     = errors.New("unsupported grant_type; only authorization_code is supported")
	ErrUnsupportedResponseType  = errors.New("unsupported response_type; only code is supported")
	ErrStateMissing             = errors.New("state parameter is required")
)
