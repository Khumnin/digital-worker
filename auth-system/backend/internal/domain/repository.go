// internal/domain/repository.go
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines all data operations on the users table.
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (*User, error)
	IncrementFailedLoginCount(ctx context.Context, id uuid.UUID) (int, error)
	ResetFailedLoginCount(ctx context.Context, id uuid.UUID) error
	SetLockedUntil(ctx context.Context, id uuid.UUID, until time.Time) error
	ListByTenant(ctx context.Context, limit, offset int, status string) ([]*User, int, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	AnonymizePII(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines all data operations on the sessions table.
type SessionRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	Create(ctx context.Context, session *Session) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeByFamilyID(ctx context.Context, familyID uuid.UUID) (int, error)
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountActiveForUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// TenantRepository defines data operations on the global public.tenants table.
type TenantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)
	FindBySchemaName(ctx context.Context, schemaName string) (*Tenant, error)
	Create(ctx context.Context, input CreateTenantInput) (*Tenant, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status TenantStatus) error
	UpdateConfig(ctx context.Context, id uuid.UUID, config TenantConfig) error
	ListActiveSchemaNames(ctx context.Context) ([]string, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Tenant, int, error)
}

// TenantCredentialRepository defines data operations on tenant_api_credentials.
type TenantCredentialRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, clientID, secretHash string) (*TenantAPICredential, error)
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*TenantAPICredential, error)
	FindByClientID(ctx context.Context, clientID string) (*TenantAPICredential, error)
	Rotate(ctx context.Context, tenantID uuid.UUID, newClientID, newSecretHash string) (*TenantAPICredential, error)
}

// AuditRepository defines all data operations on the per-tenant audit_log table.
type AuditRepository interface {
	Append(ctx context.Context, event *AuditEvent) error
	List(ctx context.Context, filter AuditFilter) ([]*AuditEvent, int, error)
	MarkArchived(ctx context.Context, ids []uuid.UUID) error
	ListForArchive(ctx context.Context, cutoff time.Time, limit int) ([]*AuditEvent, error)
	// AnonymizeActor replaces all actor_id references to userID with tombstoneID.
	// Used for GDPR erasure to decouple audit history from the deleted identity.
	AnonymizeActor(ctx context.Context, userID uuid.UUID, tombstoneID uuid.UUID) error
}

// AuditFilter defines query parameters for listing audit log entries.
type AuditFilter struct {
	EventType    *string
	ActorID      *uuid.UUID
	TargetUserID *uuid.UUID
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

// RoleRepository defines data operations on the roles and user_roles tables.
type RoleRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	ListAll(ctx context.Context) ([]*Role, error)
	Create(ctx context.Context, name, description string, module *string) (*Role, error)
	// Delete removes a role by ID. The caller is responsible for ensuring
	// the role is not a system role and is not assigned to any user before calling this.
	Delete(ctx context.Context, id uuid.UUID) error
	// IsAssignedToAnyUser returns true if the role is currently referenced
	// by at least one row in the user_roles table.
	IsAssignedToAnyUser(ctx context.Context, id uuid.UUID) (bool, error)
	AssignToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
	UnassignFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)
	// ReplaceUserRoles atomically deletes all existing roles for the user and
	// inserts the new set within a single transaction.
	ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
}

// TokenRepository defines operations on password_reset_tokens and email_verification_tokens.
type TokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, tokenHash string) error
}

// PasswordResetToken is a domain value object for password reset operations.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *PasswordResetToken) IsValid() bool {
	return !t.Used && time.Now().Before(t.ExpiresAt)
}

// EmailVerificationToken is a domain value object for email verification.
type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *EmailVerificationToken) IsValid() bool {
	return !t.Used && time.Now().Before(t.ExpiresAt)
}

// Role is the RBAC role entity scoped to a tenant.
type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	Module      *string // nil for system roles; set for module-scoped roles (e.g. "recruit")
	IsSystem    bool
	CreatedAt   time.Time
}

// AuditEvent is an immutable record of an authentication action.
type AuditEvent struct {
	ID            uuid.UUID
	EventType     EventType
	ActorID       *uuid.UUID
	ActorIP       string
	ActorUA       string
	TargetUserID  *uuid.UUID
	Metadata      map[string]interface{}
	OccurredAt    time.Time
	Archived      bool

	// Populated by List query via LEFT JOIN; nil for Append/Archive paths.
	ActorEmail  *string
	TargetEmail *string
}

// Role domain errors.
var (
	ErrRoleNotFound        = errors.New("role not found")
	ErrRoleAlreadyExists   = errors.New("role already exists in this tenant")
	ErrRoleAlreadyAssigned = errors.New("user already has this role")
	ErrRoleNotAssigned     = errors.New("user does not have this role")
	ErrSystemRole          = errors.New("cannot delete a system role")
	ErrRoleInUse           = errors.New("cannot delete a role that is assigned to users")
)

// OAuthClientRepository defines data operations on the oauth_clients table (per-tenant schema).
type OAuthClientRepository interface {
	Create(ctx context.Context, client *OAuthClient) error
	FindByClientID(ctx context.Context, clientID string) (*OAuthClient, error)
	ListByCreator(ctx context.Context, createdBy uuid.UUID) ([]*OAuthClient, error)
}

// AuthorizationCodeRepository defines data operations on the oauth_authorization_codes table.
type AuthorizationCodeRepository interface {
	Create(ctx context.Context, code *AuthorizationCode) error
	FindByCodeHash(ctx context.Context, codeHash string) (*AuthorizationCode, error)
	MarkUsed(ctx context.Context, codeHash string) error
	// DeleteByUserID removes all authorization codes issued to the given user.
	// Called on password change and GDPR erasure to invalidate outstanding codes.
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
