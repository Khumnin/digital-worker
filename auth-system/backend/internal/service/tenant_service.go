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

// TenantService handles tenant provisioning, retrieval, and lifecycle
// management.
type TenantService interface {
	ProvisionTenant(ctx context.Context, input ProvisionTenantInput) (*domain.Tenant, error)
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	ListTenants(ctx context.Context, limit, offset int) ([]*domain.Tenant, int, error)
	SuspendTenant(ctx context.Context, id string) error
	GenerateAPICredentials(ctx context.Context, tenantID string) (clientID, clientSecret string, err error)
	RotateAPICredentials(ctx context.Context, tenantID string) (clientID, clientSecret string, err error)
}

// ProvisionTenantInput carries the fields required to create a new tenant.
type ProvisionTenantInput struct {
	Name       string
	Slug       string
	AdminEmail string
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

	tenant, err := s.tenantRepo.Create(ctx, domain.CreateTenantInput{
		Name:       input.Name,
		Slug:       input.Slug,
		AdminEmail: input.AdminEmail,
		Config:     domain.DefaultTenantConfig(),
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

func (s *tenantServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
	}
}
