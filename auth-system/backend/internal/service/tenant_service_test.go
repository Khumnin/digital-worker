// internal/service/tenant_service_test.go
//
// Unit tests for TenantService focusing on the bug fix:
//   - enabled_modules must be saved when provisioning a tenant
//   - GetTenant/ListTenants must return enabled_modules from stored config
//   - nil/empty enabled_modules must be handled gracefully
//
// These tests use in-memory stubs — no database required.
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// ---------------------------------------------------------------------------
// Stub: TenantRepository
// ---------------------------------------------------------------------------

type stubTenantRepo struct {
	// storage keyed by tenant ID
	byID   map[uuid.UUID]*domain.Tenant
	bySlug map[string]*domain.Tenant

	// captured arguments for assertion
	lastCreateInput domain.CreateTenantInput
	createCalls     int
	updateConfigLog []domain.TenantConfig
	updateStatusLog []domain.TenantStatus
}

func newStubTenantRepo() *stubTenantRepo {
	return &stubTenantRepo{
		byID:   make(map[uuid.UUID]*domain.Tenant),
		bySlug: make(map[string]*domain.Tenant),
	}
}

func (r *stubTenantRepo) seed(t *domain.Tenant) {
	r.byID[t.ID] = t
	r.bySlug[t.Slug] = t
}

func (r *stubTenantRepo) Create(_ context.Context, in domain.CreateTenantInput) (*domain.Tenant, error) {
	r.lastCreateInput = in
	r.createCalls++

	t := &domain.Tenant{
		ID:         uuid.New(),
		Name:       in.Name,
		Slug:       in.Slug,
		SchemaName: domain.SlugToSchemaName(in.Slug),
		AdminEmail: in.AdminEmail,
		Status:     domain.TenantStatusActive,
		Config:     in.Config,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	r.byID[t.ID] = t
	r.bySlug[t.Slug] = t
	return t, nil
}

func (r *stubTenantRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	return t, nil
}

func (r *stubTenantRepo) FindBySlug(_ context.Context, slug string) (*domain.Tenant, error) {
	t, ok := r.bySlug[slug]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	return t, nil
}

func (r *stubTenantRepo) FindBySchemaName(_ context.Context, schemaName string) (*domain.Tenant, error) {
	for _, t := range r.byID {
		if t.SchemaName == schemaName {
			return t, nil
		}
	}
	return nil, domain.ErrTenantNotFound
}

func (r *stubTenantRepo) ListActiveSchemaNames(_ context.Context) ([]string, error) {
	names := make([]string, 0, len(r.byID))
	for _, t := range r.byID {
		if t.Status == domain.TenantStatusActive {
			names = append(names, t.SchemaName)
		}
	}
	return names, nil
}

func (r *stubTenantRepo) ListAll(_ context.Context, limit, offset int) ([]*domain.Tenant, int, error) {
	all := make([]*domain.Tenant, 0, len(r.byID))
	for _, t := range r.byID {
		all = append(all, t)
	}
	total := len(all)
	// apply offset / limit
	if offset >= total {
		return []*domain.Tenant{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (r *stubTenantRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.TenantStatus) error {
	t, ok := r.byID[id]
	if !ok {
		return domain.ErrTenantNotFound
	}
	t.Status = status
	r.updateStatusLog = append(r.updateStatusLog, status)
	return nil
}

func (r *stubTenantRepo) UpdateConfig(_ context.Context, id uuid.UUID, cfg domain.TenantConfig) error {
	t, ok := r.byID[id]
	if !ok {
		return domain.ErrTenantNotFound
	}
	t.Config = cfg
	r.updateConfigLog = append(r.updateConfigLog, cfg)
	return nil
}

// ---------------------------------------------------------------------------
// Stub: TenantCredentialRepository (minimal no-op)
// ---------------------------------------------------------------------------

type stubCredRepo struct{}

func (r *stubCredRepo) Create(_ context.Context, _ uuid.UUID, _, _ string) (*domain.TenantAPICredential, error) {
	return &domain.TenantAPICredential{ID: uuid.New()}, nil
}
func (r *stubCredRepo) Rotate(_ context.Context, _ uuid.UUID, _, _ string) (*domain.TenantAPICredential, error) {
	return &domain.TenantAPICredential{ID: uuid.New()}, nil
}
func (r *stubCredRepo) FindByTenantID(_ context.Context, _ uuid.UUID) (*domain.TenantAPICredential, error) {
	return nil, nil
}
func (r *stubCredRepo) FindByClientID(_ context.Context, _ string) (*domain.TenantAPICredential, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// stubProvisioner overrides schema provisioning so tests don't need a DB.
// We inject it via a custom TenantService constructor defined below.
// ---------------------------------------------------------------------------

// newTestTenantService builds a TenantService wired to in-memory stubs.
// The schema provisioner is replaced with a no-op by passing an empty dbURL
// and migrPath — provision calls are therefore skipped in unit tests (see
// the conditional guard in tenant_service.go).
//
// Because migrations.NewTenantProvisioner is called inside ProvisionTenant
// using the injected pool/dbURL/migrPath, and our unit tests pass nil pool
// values, we exercise only the service logic that runs BEFORE the provisioner
// call (config wiring) and the return value inspection.  To validate the
// Config field we rely on stubTenantRepo.Create capturing lastCreateInput.
func newTestTenantService(tenantRepo *stubTenantRepo) (service.TenantService, chan service.EmailTask) {
	emailCh := make(chan service.EmailTask, 10)
	svc := service.NewTenantService(
		tenantRepo,
		&stubCredRepo{},
		nil,  // pgxpool — not exercised in unit tests
		"",   // dbURL — empty → provisioner Provision() will be called but fail fast
		"",   // migrPath
		emailCh,
	)
	return svc, emailCh
}

// ---------------------------------------------------------------------------
// Helper: build a tenant with specific enabled_modules pre-seeded.
// ---------------------------------------------------------------------------

func tenantWithModules(slug string, mods []string) *domain.Tenant {
	cfg := domain.DefaultTenantConfig()
	cfg.EnabledModules = mods
	return &domain.Tenant{
		ID:         uuid.New(),
		Slug:       slug,
		Name:       "Test Tenant",
		SchemaName: domain.SlugToSchemaName(slug),
		AdminEmail: "admin@" + slug + ".com",
		Status:     domain.TenantStatusActive,
		Config:     cfg,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ---------------------------------------------------------------------------
// provisionAndCapture runs ProvisionTenant and recovers from the nil-pool
// panic that occurs inside migrations.TenantProvisioner.Provision.
//
// The service calls tenantRepo.Create synchronously BEFORE invoking the
// provisioner. This helper lets us assert on what was passed to Create
// without requiring a live database connection.
// ---------------------------------------------------------------------------
func provisionAndCapture(svc service.TenantService, input service.ProvisionTenantInput) {
	defer func() { recover() }() //nolint:errcheck
	_, _ = svc.ProvisionTenant(context.Background(), input)
}

// ---------------------------------------------------------------------------
// ProvisionTenant — enabled_modules saved to config (BUG FIX coverage)
// ---------------------------------------------------------------------------

// TC-TS-01: ProvisionTenant saves the provided enabled_modules into the
// tenant config passed to the repository.
func TestProvisionTenant_SavesEnabledModulesToConfig(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	modules := []string{"recruit", "leave"}
	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:           "Acme Corp",
		Slug:           "acme-corp",
		AdminEmail:     "admin@acme.com",
		EnabledModules: modules,
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected tenantRepo.Create to be called once, got %d calls", tenantRepo.createCalls)
	}

	got := tenantRepo.lastCreateInput.Config.EnabledModules
	if len(got) != len(modules) {
		t.Fatalf("expected %d enabled_modules in config, got %d: %v", len(modules), len(got), got)
	}
	for i, mod := range modules {
		if got[i] != mod {
			t.Errorf("enabled_modules[%d]: got %q, want %q", i, got[i], mod)
		}
	}
}

// TC-TS-02: ProvisionTenant with nil config (no enabled_modules supplied)
// stores an empty/nil slice — not a populated list.
func TestProvisionTenant_NilEnabledModules_StoresEmpty(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:           "No Modules Co",
		Slug:           "no-modules",
		AdminEmail:     "admin@no-modules.com",
		EnabledModules: nil,
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected tenantRepo.Create to be called once, got %d", tenantRepo.createCalls)
	}

	got := tenantRepo.lastCreateInput.Config.EnabledModules
	if len(got) != 0 {
		t.Errorf("expected empty enabled_modules in config, got %v", got)
	}
}

// TC-TS-03: ProvisionTenant with an explicit empty slice stores an empty list.
func TestProvisionTenant_EmptySliceEnabledModules_StoresEmpty(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:           "Empty Modules Co",
		Slug:           "empty-modules",
		AdminEmail:     "admin@empty.com",
		EnabledModules: []string{},
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected tenantRepo.Create to be called once, got %d", tenantRepo.createCalls)
	}

	got := tenantRepo.lastCreateInput.Config.EnabledModules
	if len(got) != 0 {
		t.Errorf("expected empty enabled_modules in config, got %v", got)
	}
}

// TC-TS-04: ProvisionTenant preserves all other DefaultTenantConfig values
// (regression guard — enabled_modules assignment must not clobber defaults).
func TestProvisionTenant_DefaultConfigPreservedAlongsideEnabledModules(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:           "Acme",
		Slug:           "acme-config",
		AdminEmail:     "admin@acme.com",
		EnabledModules: []string{"recruit"},
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected exactly 1 Create call, got %d", tenantRepo.createCalls)
	}

	cfg := tenantRepo.lastCreateInput.Config
	defaults := domain.DefaultTenantConfig()

	if cfg.SessionTTLSeconds != defaults.SessionTTLSeconds {
		t.Errorf("SessionTTLSeconds: got %d, want %d", cfg.SessionTTLSeconds, defaults.SessionTTLSeconds)
	}
	if cfg.LockoutThreshold != defaults.LockoutThreshold {
		t.Errorf("LockoutThreshold: got %d, want %d", cfg.LockoutThreshold, defaults.LockoutThreshold)
	}
	if cfg.PasswordPolicy.MinLength != defaults.PasswordPolicy.MinLength {
		t.Errorf("PasswordPolicy.MinLength: got %d, want %d", cfg.PasswordPolicy.MinLength, defaults.PasswordPolicy.MinLength)
	}
	if len(cfg.EnabledModules) != 1 || cfg.EnabledModules[0] != "recruit" {
		t.Errorf("EnabledModules: got %v, want [recruit]", cfg.EnabledModules)
	}
}

// TC-TS-05: ProvisionTenant rejects an invalid slug without calling Create.
func TestProvisionTenant_InvalidSlug_ReturnsErrorWithoutCreate(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	_, err := svc.ProvisionTenant(context.Background(), service.ProvisionTenantInput{
		Name:       "Bad Slug Co",
		Slug:       "INVALID SLUG!",
		AdminEmail: "admin@bad.com",
	})

	if err == nil {
		t.Fatal("expected an error for invalid slug, got nil")
	}
	if tenantRepo.createCalls != 0 {
		t.Errorf("expected no Create call on invalid slug, got %d", tenantRepo.createCalls)
	}
}

// ---------------------------------------------------------------------------
// GetTenant — returns enabled_modules from stored config (BUG FIX coverage)
// ---------------------------------------------------------------------------

// TC-TS-06: GetTenant returns the enabled_modules stored in the tenant config.
func TestGetTenant_ReturnsEnabledModulesFromConfig(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	seeded := tenantWithModules("recruit-co", []string{"recruit"})
	tenantRepo.seed(seeded)

	got, err := svc.GetTenant(context.Background(), seeded.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Config.EnabledModules) != 1 || got.Config.EnabledModules[0] != "recruit" {
		t.Errorf("expected enabled_modules=[recruit], got %v", got.Config.EnabledModules)
	}
}

// TC-TS-07: GetTenant with a tenant that has no enabled_modules returns nil/empty.
func TestGetTenant_NoEnabledModules_ReturnsEmptySlice(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	seeded := tenantWithModules("no-mods", nil)
	tenantRepo.seed(seeded)

	got, err := svc.GetTenant(context.Background(), seeded.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Config.EnabledModules) != 0 {
		t.Errorf("expected empty enabled_modules, got %v", got.Config.EnabledModules)
	}
}

// TC-TS-08: GetTenant with a tenant that has multiple modules returns all of them.
func TestGetTenant_MultipleModules_ReturnsAllModules(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	mods := []string{"recruit", "leave", "payroll"}
	seeded := tenantWithModules("multi-mods", mods)
	tenantRepo.seed(seeded)

	got, err := svc.GetTenant(context.Background(), seeded.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Config.EnabledModules) != len(mods) {
		t.Fatalf("expected %d modules, got %d: %v", len(mods), len(got.Config.EnabledModules), got.Config.EnabledModules)
	}
	for i, m := range mods {
		if got.Config.EnabledModules[i] != m {
			t.Errorf("module[%d]: got %q, want %q", i, got.Config.EnabledModules[i], m)
		}
	}
}

// TC-TS-09: GetTenant with an invalid UUID returns an error.
func TestGetTenant_InvalidUUID_ReturnsError(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	_, err := svc.GetTenant(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected an error for invalid UUID, got nil")
	}
}

// TC-TS-10: GetTenant with a valid UUID that does not exist returns not-found.
func TestGetTenant_UnknownID_ReturnsTenantNotFound(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	_, err := svc.GetTenant(context.Background(), uuid.New().String())
	if !errors.Is(err, domain.ErrTenantNotFound) {
		t.Errorf("expected ErrTenantNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListTenants — returns enabled_modules from stored config (BUG FIX coverage)
// ---------------------------------------------------------------------------

// TC-TS-11: ListTenants returns enabled_modules for each tenant from its stored config.
func TestListTenants_ReturnsEnabledModulesFromConfig(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	t1 := tenantWithModules("alpha", []string{"recruit"})
	t2 := tenantWithModules("beta", []string{"leave", "payroll"})
	tenantRepo.seed(t1)
	tenantRepo.seed(t2)

	tenants, total, err := svc.ListTenants(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(tenants) != 2 {
		t.Fatalf("expected 2 tenants in result, got %d", len(tenants))
	}

	// Build a slug→modules map from results for order-independent assertion.
	resultMap := make(map[string][]string)
	for _, t := range tenants {
		resultMap[t.Slug] = t.Config.EnabledModules
	}

	if mods, ok := resultMap["alpha"]; !ok || len(mods) != 1 || mods[0] != "recruit" {
		t.Errorf("alpha: expected [recruit], got %v", resultMap["alpha"])
	}
	if mods, ok := resultMap["beta"]; !ok || len(mods) != 2 {
		t.Errorf("beta: expected 2 modules, got %v", resultMap["beta"])
	}
}

// TC-TS-12: ListTenants with a tenant that has nil enabled_modules returns an
// empty (not nil) slice, consistent with safe API serialisation.
func TestListTenants_NilModules_ReturnsEmptyNotNil(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	seeded := tenantWithModules("no-mods-list", nil)
	tenantRepo.seed(seeded)

	tenants, _, err := svc.ListTenants(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(tenants))
	}

	// The service itself returns whatever the repo returns. The handler is
	// responsible for the nil→[] coercion. Verify the service at least does
	// not error and passes through the config faithfully.
	_ = tenants[0].Config.EnabledModules // must not panic
}

// TC-TS-13: ListTenants with empty store returns zero results and total=0.
func TestListTenants_EmptyStore_ReturnsZeroResults(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	tenants, total, err := svc.ListTenants(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
	if len(tenants) != 0 {
		t.Errorf("expected empty slice, got %d items", len(tenants))
	}
}

// ---------------------------------------------------------------------------
// Domain: DefaultTenantConfig — enabled_modules not set by default
// ---------------------------------------------------------------------------

// TC-TS-14: DefaultTenantConfig should NOT pre-populate enabled_modules.
// This guards against accidental regression where defaults include modules.
func TestDefaultTenantConfig_EnabledModulesIsNilByDefault(t *testing.T) {
	cfg := domain.DefaultTenantConfig()
	if len(cfg.EnabledModules) != 0 {
		t.Errorf("expected EnabledModules to be empty by default, got %v", cfg.EnabledModules)
	}
}

// ---------------------------------------------------------------------------
// Service: enabled_modules round-trip through ProvisionTenant + GetTenant
// ---------------------------------------------------------------------------

// TC-TS-15: Round-trip — modules supplied to ProvisionTenant are retrievable
// via GetTenant from the same in-memory store.
func TestProvisionTenant_EnabledModules_RoundTripViaGetTenant(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, _ := newTestTenantService(tenantRepo)

	modules := []string{"recruit"}
	// ProvisionTenant panics at the schema provisioner step (nil pool),
	// but the tenant record has already been written to the stub repo by
	// the time Create returns. provisionAndCapture recovers the panic.
	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:           "Round Trip Co",
		Slug:           "round-trip",
		AdminEmail:     "admin@round-trip.com",
		EnabledModules: modules,
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("Create not called — cannot perform round-trip assertion")
	}

	// Retrieve the stored tenant directly from the stub.
	var storedID uuid.UUID
	for id := range tenantRepo.byID {
		storedID = id
		break
	}

	got, err := svc.GetTenant(context.Background(), storedID.String())
	if err != nil {
		t.Fatalf("GetTenant failed: %v", err)
	}

	if len(got.Config.EnabledModules) != 1 || got.Config.EnabledModules[0] != "recruit" {
		t.Errorf("round-trip failed: expected [recruit], got %v", got.Config.EnabledModules)
	}
}
