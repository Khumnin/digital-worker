// internal/domain/tenant.go
package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the operational state of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// Tenant is the top-level multi-tenancy entity stored in the global public schema.
type Tenant struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	SchemaName  string
	AdminEmail  string
	Status      TenantStatus
	Config      TenantConfig
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// PasswordPolicy defines the password complexity rules configurable per tenant.
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireNumber    bool `json:"require_number"`
	RequireSpecial   bool `json:"require_special"`
}

// TenantConfig holds all tenant-level runtime configuration.
type TenantConfig struct {
	SessionTTLSeconds           int            `json:"session_ttl_seconds"`
	SlidingSessionEnabled       bool           `json:"sliding_session_enabled"`
	LockoutThreshold            int            `json:"lockout_threshold"`
	LockoutDurationSeconds      int            `json:"lockout_duration_seconds"`
	PasswordPolicy              PasswordPolicy `json:"password_policy"`
	GoogleClientID              string         `json:"google_client_id,omitempty"`
	GoogleClientSecret          string         `json:"google_client_secret,omitempty"`
	AllowedCORSOrigins          []string       `json:"allowed_cors_origins,omitempty"`
	MFARequired                 bool           `json:"mfa_required"`
}

// DefaultTenantConfig returns a secure-by-default tenant configuration.
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		SessionTTLSeconds:      86400,
		SlidingSessionEnabled:  false,
		LockoutThreshold:       5,
		LockoutDurationSeconds: 900,
		PasswordPolicy: PasswordPolicy{
			MinLength:        12,
			RequireUppercase: true,
			RequireNumber:    true,
			RequireSpecial:   true,
		},
	}
}

var slugRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,48}[a-z0-9]$`)
var schemaNameRegexp = regexp.MustCompile(`^tenant_[a-z0-9_]{1,50}$`)

// SlugToSchemaName converts a tenant slug to a PostgreSQL schema name.
func SlugToSchemaName(slug string) string {
	safe := regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(slug, "_")
	return "tenant_" + safe
}

// ValidateSlug returns an error if the slug does not meet format requirements.
func ValidateSlug(slug string) error {
	if !slugRegexp.MatchString(slug) {
		return errors.New("slug must be 3-50 characters: lowercase letters, numbers, hyphens only; cannot start or end with a hyphen")
	}
	return nil
}

// IsValidSchemaName validates a schema name before use in SET search_path.
func IsValidSchemaName(name string) bool {
	return schemaNameRegexp.MatchString(name)
}

// Tenant domain errors.
var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant with this slug already exists")
	ErrTenantSuspended     = errors.New("tenant is suspended")
	ErrInvalidSlug         = errors.New("invalid tenant slug format")
	ErrInvalidSchemaName   = errors.New("invalid schema name format")
)

// CreateTenantInput carries validated input for provisioning a new tenant.
type CreateTenantInput struct {
	Name       string
	Slug       string
	AdminEmail string
	Config     TenantConfig
}
