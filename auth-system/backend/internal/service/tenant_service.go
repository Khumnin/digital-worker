// internal/service/tenant_service.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/infrastructure/migrations"
	"tigersoft/auth-system/pkg/crypto"
)

// UpdateTenantSettingsInput carries the fields that can be updated via PUT /admin/tenant.
// All fields are optional — only non-nil/provided fields are applied.
type UpdateTenantSettingsInput struct {
	MFARequired          *bool
	SessionDurationHours *int
	AllowedDomains       []string
}

// TenantService handles tenant provisioning, retrieval, and lifecycle
// management.
type TenantService interface {
	ProvisionTenant(ctx context.Context, input ProvisionTenantInput) (*domain.Tenant, error)
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	ListTenants(ctx context.Context, limit, offset int) ([]*domain.Tenant, int, error)
	SuspendTenant(ctx context.Context, id string) error
	ActivateTenant(ctx context.Context, id string) error
	GenerateAPICredentials(ctx context.Context, tenantID string) (clientID, clientSecret string, err error)
	RotateAPICredentials(ctx context.Context, tenantID string) (clientID, clientSecret string, err error)
	// UpdateMFARequirement toggles the MFA enforcement flag for the tenant.
	UpdateMFARequirement(ctx context.Context, tenantID string, required bool) error
	// UpdateTenantSettings applies a partial update to the calling admin's tenant config.
	UpdateTenantSettings(ctx context.Context, tenantSlug string, input UpdateTenantSettingsInput) (*domain.Tenant, error)
}

// ProvisionTenantInput carries the fields required to create a new tenant.
type ProvisionTenantInput struct {
	Name           string
	Slug           string
	AdminEmail     string
	EnabledModules []string
}

type tenantServiceImpl struct {
	tenantRepo     domain.TenantRepository
	credRepo       domain.TenantCredentialRepository
	pool           *pgxpool.Pool
	dbURL          string
	tenantMigrPath string
	emailCh        chan<- EmailTask
}

// NewTenantService constructs a TenantService with all dependencies injected.
func NewTenantService(
	tenantRepo domain.TenantRepository,
	credRepo domain.TenantCredentialRepository,
	pool *pgxpool.Pool,
	dbURL string,
	tenantMigrPath string,
	emailCh chan<- EmailTask,
) TenantService {
	return &tenantServiceImpl{
		tenantRepo:     tenantRepo,
		credRepo:       credRepo,
		pool:           pool,
		dbURL:          dbURL,
		tenantMigrPath: tenantMigrPath,
		emailCh:        emailCh,
	}
}

// ProvisionTenant validates the slug, creates the tenant record in the global
// schema, then runs per-tenant migrations to create the isolated schema.
func (s *tenantServiceImpl) ProvisionTenant(ctx context.Context, input ProvisionTenantInput) (*domain.Tenant, error) {
	if err := domain.ValidateSlug(input.Slug); err != nil {
		return nil, fmt.Errorf("invalid slug: %w", err)
	}

	cfg := domain.DefaultTenantConfig()
	cfg.EnabledModules = input.EnabledModules

	tenant, err := s.tenantRepo.Create(ctx, domain.CreateTenantInput{
		Name:       input.Name,
		Slug:       input.Slug,
		AdminEmail: input.AdminEmail,
		Config:     cfg,
	})
	if err != nil {
		return nil, fmt.Errorf("create tenant record: %w", err)
	}

	provisioner := migrations.NewTenantProvisioner(s.pool, s.dbURL, s.tenantMigrPath)
	if err := provisioner.Provision(ctx, tenant.SchemaName); err != nil {
		return nil, fmt.Errorf("provision tenant schema %q: %w", tenant.SchemaName, err)
	}

	s.enqueueEmail(EmailTask{
		Type:    EmailTypeInvitation,
		ToEmail: tenant.AdminEmail,
		ToName:  tenant.Name,
		Extra: map[string]interface{}{
			"tenant_name": tenant.Name,
			"tenant_slug": tenant.Slug,
		},
	})

	return tenant, nil
}

// GetTenant returns a single tenant by its UUID string.
func (s *tenantServiceImpl) GetTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("find tenant: %w", err)
	}

	return tenant, nil
}

// GetTenantBySlug returns a single tenant by its slug.
func (s *tenantServiceImpl) GetTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("find tenant by slug: %w", err)
	}
	return tenant, nil
}

// ListTenants returns a paginated list of all tenants with the total count.
func (s *tenantServiceImpl) ListTenants(ctx context.Context, limit, offset int) ([]*domain.Tenant, int, error) {
	tenants, total, err := s.tenantRepo.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}

	return tenants, total, nil
}

// SuspendTenant moves a tenant into the suspended state, preventing logins.
func (s *tenantServiceImpl) SuspendTenant(ctx context.Context, id string) error {
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	if err := s.tenantRepo.UpdateStatus(ctx, tenantID, domain.TenantStatusSuspended); err != nil {
		return fmt.Errorf("suspend tenant: %w", err)
	}

	return nil
}

// ActivateTenant moves a tenant into the active state.
func (s *tenantServiceImpl) ActivateTenant(ctx context.Context, id string) error {
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	if err := s.tenantRepo.UpdateStatus(ctx, tenantID, domain.TenantStatusActive); err != nil {
		return fmt.Errorf("activate tenant: %w", err)
	}

	return nil
}

// GenerateAPICredentials creates a new client_id + client_secret pair for the
// tenant. The secret is returned once in plaintext and never again.
func (s *tenantServiceImpl) GenerateAPICredentials(ctx context.Context, tenantIDStr string) (string, string, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	clientID := uuid.New().String()
	rawSecret, secretHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return "", "", fmt.Errorf("generate client secret: %w", err)
	}

	if _, err := s.credRepo.Create(ctx, tenantID, clientID, secretHash); err != nil {
		return "", "", fmt.Errorf("store api credential: %w", err)
	}

	return clientID, rawSecret, nil
}

// RotateAPICredentials revokes all existing credentials for the tenant and
// issues a new client_id + client_secret pair. Old credentials are immediately
// invalidated.
func (s *tenantServiceImpl) RotateAPICredentials(ctx context.Context, tenantIDStr string) (string, string, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid tenant ID: %w", err)
	}

	newClientID := uuid.New().String()
	rawSecret, secretHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return "", "", fmt.Errorf("generate client secret: %w", err)
	}

	if _, err := s.credRepo.Rotate(ctx, tenantID, newClientID, secretHash); err != nil {
		return "", "", fmt.Errorf("rotate api credential: %w", err)
	}

	return newClientID, rawSecret, nil
}

// UpdateMFARequirement sets the mfa_required flag on the tenant's stored config.
// Loads the current config, toggles the flag, and persists the full config JSON
// so all other fields remain unchanged.
func (s *tenantServiceImpl) UpdateMFARequirement(ctx context.Context, tenantIDStr string, required bool) error {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("find tenant for MFA update: %w", err)
	}

	tenant.Config.MFARequired = required

	if err := s.tenantRepo.UpdateConfig(ctx, tenantID, tenant.Config); err != nil {
		return fmt.Errorf("persist MFA requirement change: %w", err)
	}

	return nil
}

// UpdateTenantSettings applies a partial update to the tenant config identified by slug.
// Only fields provided in the input are mutated; others retain their current values.
func (s *tenantServiceImpl) UpdateTenantSettings(ctx context.Context, tenantSlug string, input UpdateTenantSettingsInput) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("find tenant by slug: %w", err)
	}

	if input.MFARequired != nil {
		tenant.Config.MFARequired = *input.MFARequired
	}
	if input.SessionDurationHours != nil {
		tenant.Config.SessionTTLSeconds = *input.SessionDurationHours * 3600
	}
	if input.AllowedDomains != nil {
		tenant.Config.AllowedCORSOrigins = input.AllowedDomains
	}

	if err := s.tenantRepo.UpdateConfig(ctx, tenant.ID, tenant.Config); err != nil {
		return nil, fmt.Errorf("persist tenant settings: %w", err)
	}

	// Reload to get updated_at from DB.
	updated, err := s.tenantRepo.FindByID(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("reload tenant after update: %w", err)
	}

	return updated, nil
}

func (s *tenantServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
	}
}
