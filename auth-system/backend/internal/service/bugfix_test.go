// internal/service/bugfix_test.go
//
// Additional unit tests covering the three bug fixes and associated warning
// fixes merged in Sprint 3:
//
//   Bug 1  — ProvisionTenant: ToName must be AdminEmail, not tenant.Name
//   Bug 2  — ProvisionTenant: TenantName / TenantSlug set on EmailTask struct
//   Bug 3  — ListUsers / ReplaceUserRoles: UserWithRoles type used in handlers
//   W-4    — ReplaceUserRoles: actorID threaded through to repo
//
// Stub types (stubUserRepo, stubRoleRepo, etc.) are defined in admin_service_test.go
// and tenant_service_test.go; this file reuses them — no redeclaration needed.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// ===========================================================================
// Bug 1 + Bug 2 — ProvisionTenant email task fields
// ===========================================================================

// TC-BF-01: ProvisionTenant creates an EmailTask whose ToName is the
// AdminEmail address, not the tenant's display name.
// Before the fix: ToName was set to tenant.Name ("TigerSoft Co., Ltd").
// After  the fix: ToName is set to tenant.AdminEmail.
func TestProvisionTenant_EmailTask_ToName_IsAdminEmail_NotTenantName(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, emailCh := newTestTenantService(tenantRepo)

	const adminEmail = "alice@acme.com"
	const tenantName = "Acme Corporation"

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:       tenantName,
		Slug:       "acme-email",
		AdminEmail: adminEmail,
	})

	// The stub repo must have been called exactly once so a tenant exists.
	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected 1 Create call, got %d — cannot assert on email task", tenantRepo.createCalls)
	}

	// Drain the email channel — the task should be present.
	var task service.EmailTask
	select {
	case task = <-emailCh:
	default:
		t.Fatal("expected an EmailTask to be enqueued, channel was empty")
	}

	// Bug 1 assertion: ToName must NOT equal the company/tenant name.
	if task.ToName == tenantName {
		t.Errorf("Bug 1 regression: ToName is %q (tenant name); expected AdminEmail %q", task.ToName, adminEmail)
	}

	// ToName must be the admin email address.
	if task.ToName != adminEmail {
		t.Errorf("ToName: got %q, want %q (admin email)", task.ToName, adminEmail)
	}
}

// TC-BF-02: ProvisionTenant creates an EmailTask whose TenantName field is
// set to the tenant's display name on the struct (not via Extra map).
// Before the fix: TenantName was absent from the struct; worker read it as "".
// After  the fix: TenantName = tenant.Name.
func TestProvisionTenant_EmailTask_TenantName_SetOnStruct(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, emailCh := newTestTenantService(tenantRepo)

	const tenantName = "Global Corp Ltd"

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:       tenantName,
		Slug:       "global-corp",
		AdminEmail: "ceo@global.com",
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected 1 Create call, got %d", tenantRepo.createCalls)
	}

	var task service.EmailTask
	select {
	case task = <-emailCh:
	default:
		t.Fatal("expected an EmailTask to be enqueued, channel was empty")
	}

	// Bug 2 assertion: TenantName must match the name passed to ProvisionTenant.
	if task.TenantName != tenantName {
		t.Errorf("Bug 2: TenantName: got %q, want %q", task.TenantName, tenantName)
	}
}

// TC-BF-03: ProvisionTenant creates an EmailTask whose TenantSlug field is
// set to the tenant's slug on the struct (not via Extra map).
func TestProvisionTenant_EmailTask_TenantSlug_SetOnStruct(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, emailCh := newTestTenantService(tenantRepo)

	const slug = "beta-org"

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:       "Beta Organisation",
		Slug:       slug,
		AdminEmail: "admin@beta.org",
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected 1 Create call, got %d", tenantRepo.createCalls)
	}

	var task service.EmailTask
	select {
	case task = <-emailCh:
	default:
		t.Fatal("expected an EmailTask to be enqueued, channel was empty")
	}

	// Bug 2 assertion: TenantSlug must match the slug passed to ProvisionTenant.
	if task.TenantSlug != slug {
		t.Errorf("Bug 2: TenantSlug: got %q, want %q", task.TenantSlug, slug)
	}
}

// TC-BF-04: ProvisionTenant EmailTask type must be EmailTypeInvitation.
// Regression guard — ensures the correct template is selected by the worker.
func TestProvisionTenant_EmailTask_TypeIsInvitation(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, emailCh := newTestTenantService(tenantRepo)

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:       "Type Check Co",
		Slug:       "type-check",
		AdminEmail: "admin@typecheck.com",
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected 1 Create call, got %d", tenantRepo.createCalls)
	}

	var task service.EmailTask
	select {
	case task = <-emailCh:
	default:
		t.Fatal("expected an EmailTask to be enqueued, channel was empty")
	}

	if task.Type != service.EmailTypeInvitation {
		t.Errorf("EmailTask.Type: got %q, want %q", task.Type, service.EmailTypeInvitation)
	}
}

// TC-BF-05: ProvisionTenant EmailTask ToEmail must be the AdminEmail address.
// Ensures the invitation is dispatched to the right recipient.
func TestProvisionTenant_EmailTask_ToEmail_IsAdminEmail(t *testing.T) {
	tenantRepo := newStubTenantRepo()
	svc, emailCh := newTestTenantService(tenantRepo)

	const adminEmail = "boss@corp.io"

	provisionAndCapture(svc, service.ProvisionTenantInput{
		Name:       "Corp IO",
		Slug:       "corp-io",
		AdminEmail: adminEmail,
	})

	if tenantRepo.createCalls != 1 {
		t.Fatalf("expected 1 Create call, got %d", tenantRepo.createCalls)
	}

	var task service.EmailTask
	select {
	case task = <-emailCh:
	default:
		t.Fatal("expected an EmailTask to be enqueued, channel was empty")
	}

	if task.ToEmail != adminEmail {
		t.Errorf("EmailTask.ToEmail: got %q, want %q", task.ToEmail, adminEmail)
	}
}

// ===========================================================================
// Bug 3 — ListUsers returns []*UserWithRoles with roles populated
// ===========================================================================

// recordingRoleRepo wraps stubRoleRepo to capture GetUserRolesBatch calls
// and return configurable role data.
type recordingRoleRepo struct {
	stubRoleRepo
	batchCalled bool
	rolesResult map[uuid.UUID][]*domain.Role
	// findByNameResult is returned for FindByName (used by ReplaceUserRoles).
	findByNameResult *domain.Role
	// replaceCallArgs captures arguments to ReplaceUserRoles.
	replaceCallArgs []replaceCallArg
}

type replaceCallArg struct {
	userID  uuid.UUID
	roleIDs []uuid.UUID
	actorID uuid.UUID
}

func (r *recordingRoleRepo) GetUserRolesBatch(_ context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*domain.Role, error) {
	r.batchCalled = true
	if r.rolesResult != nil {
		return r.rolesResult, nil
	}
	return map[uuid.UUID][]*domain.Role{}, nil
}

func (r *recordingRoleRepo) FindByName(_ context.Context, _ string) (*domain.Role, error) {
	if r.findByNameResult != nil {
		return r.findByNameResult, nil
	}
	return nil, domain.ErrRoleNotFound
}

func (r *recordingRoleRepo) GetUserRoles(_ context.Context, userID uuid.UUID) ([]*domain.Role, error) {
	if r.rolesResult != nil {
		return r.rolesResult[userID], nil
	}
	return nil, nil
}

func (r *recordingRoleRepo) ReplaceUserRoles(_ context.Context, userID uuid.UUID, roleIDs []uuid.UUID, actorID uuid.UUID) error {
	r.replaceCallArgs = append(r.replaceCallArgs, replaceCallArg{
		userID:  userID,
		roleIDs: roleIDs,
		actorID: actorID,
	})
	return nil
}

// listingUserRepo returns a configurable set of users for ListByTenant.
type listingUserRepo struct {
	stubUserRepo
	users []*domain.User
}

func (r *listingUserRepo) ListByTenant(_ context.Context, _, _ int, _ string) ([]*domain.User, int, error) {
	return r.users, len(r.users), nil
}

// newAdminServiceWithRoleRepo builds an AdminService with a custom role repo
// for testing Bug 3 and W-4.
func newAdminServiceWithRoleRepo(userRepo domain.UserRepository, roleRepo domain.RoleRepository) service.AdminService {
	emailCh := make(chan service.EmailTask, 10)
	return service.NewAdminService(
		userRepo,
		&stubSessionRepo{},
		roleRepo,
		&stubAuditRepo{},
		&stubMFARepo{},
		&stubSocialAccountRepo{},
		&stubAuthCodeRepo{},
		nil,
		emailCh,
	)
}

// TC-BF-06: ListUsers returns []*UserWithRoles with roles populated when
// GetUserRolesBatch returns data for the listed users.
func TestListUsers_ReturnsUsersWithRolesPopulated(t *testing.T) {
	userID := uuid.New()
	modPtr := func(s string) *string { return &s }

	users := []*domain.User{
		{
			ID:        userID,
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Example",
			Status:    domain.UserStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	adminRole := &domain.Role{ID: uuid.New(), Name: "admin", IsSystem: true}
	recruitRole := &domain.Role{ID: uuid.New(), Name: "recruiter", Module: modPtr("recruit"), IsSystem: false}

	roleRepo := &recordingRoleRepo{
		rolesResult: map[uuid.UUID][]*domain.Role{
			userID: {adminRole, recruitRole},
		},
	}

	userRepo := &listingUserRepo{
		stubUserRepo: *newStubUserRepo(),
		users:        users,
	}

	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	result, total, err := svc.ListUsers(context.Background(), 20, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total: got %d, want 1", total)
	}
	if len(result) != 1 {
		t.Fatalf("result length: got %d, want 1", len(result))
	}

	uwr := result[0]
	if uwr.User == nil {
		t.Fatal("UserWithRoles.User is nil")
	}
	if uwr.User.ID != userID {
		t.Errorf("User.ID: got %s, want %s", uwr.User.ID, userID)
	}

	// Bug 3 assertion: system roles must be populated.
	if len(uwr.SystemRoles) != 1 || uwr.SystemRoles[0] != "admin" {
		t.Errorf("SystemRoles: got %v, want [admin]", uwr.SystemRoles)
	}

	// Bug 3 assertion: module roles must be populated.
	recruitRoles, ok := uwr.ModuleRoles["recruit"]
	if !ok || len(recruitRoles) != 1 || recruitRoles[0] != "recruiter" {
		t.Errorf("ModuleRoles[recruit]: got %v, want [recruiter]", uwr.ModuleRoles)
	}

	// GetUserRolesBatch must have been called (not N+1 individual queries).
	if !roleRepo.batchCalled {
		t.Error("expected GetUserRolesBatch to be called for batch role resolution")
	}
}

// TC-BF-07: ListUsers with empty user list returns empty result and zero total.
// Regression guard — must not panic on empty slice.
func TestListUsers_EmptyResult_ReturnsZeroItems(t *testing.T) {
	userRepo := &listingUserRepo{
		stubUserRepo: *newStubUserRepo(),
		users:        []*domain.User{},
	}
	roleRepo := &recordingRoleRepo{}
	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	result, total, err := svc.ListUsers(context.Background(), 20, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("total: got %d, want 0", total)
	}
	if len(result) != 0 {
		t.Errorf("result length: got %d, want 0", len(result))
	}
}

// TC-BF-08: ListUsers — user with no roles gets empty SystemRoles and
// ModuleRoles (not nil) — safe for JSON serialisation.
func TestListUsers_UserWithNoRoles_ReturnsEmptyRoleSlices(t *testing.T) {
	userID := uuid.New()
	users := []*domain.User{
		{ID: userID, Email: "noroles@example.com", Status: domain.UserStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	userRepo := &listingUserRepo{stubUserRepo: *newStubUserRepo(), users: users}
	roleRepo := &recordingRoleRepo{} // returns empty map

	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	result, _, err := svc.ListUsers(context.Background(), 20, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("result length: got %d, want 1", len(result))
	}

	uwr := result[0]
	if uwr.SystemRoles == nil {
		t.Error("SystemRoles should be an empty slice, not nil")
	}
	if uwr.ModuleRoles == nil {
		t.Error("ModuleRoles should be an empty map, not nil")
	}
}

// ===========================================================================
// W-4 — ReplaceUserRoles threads actorID to the repository
// ===========================================================================

// TC-BF-09: ReplaceUserRoles passes the provided actorID to
// roleRepo.ReplaceUserRoles. Before W-4 the actor was always uuid.Nil.
func TestReplaceUserRoles_ThreadsActorIDToRepo(t *testing.T) {
	actorID := uuid.New()
	targetUserID := uuid.New()
	roleID := uuid.New()

	userRepo := newStubUserRepo()
	userRepo.seed(&domain.User{
		ID:        targetUserID,
		Email:     "target@example.com",
		Status:    domain.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	roleRepo := &recordingRoleRepo{
		findByNameResult: &domain.Role{ID: roleID, Name: "admin", IsSystem: true},
	}

	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	_, err := svc.ReplaceUserRoles(
		context.Background(),
		targetUserID.String(),
		[]string{"admin"},
		map[string][]string{},
		actorID.String(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roleRepo.replaceCallArgs) != 1 {
		t.Fatalf("expected ReplaceUserRoles to be called once, got %d", len(roleRepo.replaceCallArgs))
	}

	call := roleRepo.replaceCallArgs[0]

	// W-4 assertion: actorID must match the actor passed by the service layer.
	if call.actorID != actorID {
		t.Errorf("actorID threaded to repo: got %s, want %s", call.actorID, actorID)
	}

	// Sanity: userID and roleID correctly forwarded.
	if call.userID != targetUserID {
		t.Errorf("userID: got %s, want %s", call.userID, targetUserID)
	}
	if len(call.roleIDs) != 1 || call.roleIDs[0] != roleID {
		t.Errorf("roleIDs: got %v, want [%s]", call.roleIDs, roleID)
	}
}

// TC-BF-10: ReplaceUserRoles with empty actorID (unauthed call) passes
// uuid.Nil to the repo and does not error.
func TestReplaceUserRoles_EmptyActorID_PassesNilUUIDToRepo(t *testing.T) {
	targetUserID := uuid.New()
	roleID := uuid.New()

	userRepo := newStubUserRepo()
	userRepo.seed(&domain.User{
		ID:        targetUserID,
		Email:     "target2@example.com",
		Status:    domain.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	roleRepo := &recordingRoleRepo{
		findByNameResult: &domain.Role{ID: roleID, Name: "user", IsSystem: true},
	}

	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	_, err := svc.ReplaceUserRoles(
		context.Background(),
		targetUserID.String(),
		[]string{"user"},
		map[string][]string{},
		"", // empty actorID
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(roleRepo.replaceCallArgs) != 1 {
		t.Fatalf("expected 1 ReplaceUserRoles call, got %d", len(roleRepo.replaceCallArgs))
	}

	call := roleRepo.replaceCallArgs[0]
	if call.actorID != uuid.Nil {
		t.Errorf("expected actorID=uuid.Nil for empty input, got %s", call.actorID)
	}
}

// TC-BF-11: ReplaceUserRoles with an unknown role name returns an error and
// does not call roleRepo.ReplaceUserRoles.
func TestReplaceUserRoles_UnknownRole_ReturnsError(t *testing.T) {
	targetUserID := uuid.New()

	userRepo := newStubUserRepo()
	userRepo.seed(&domain.User{
		ID:        targetUserID,
		Email:     "target3@example.com",
		Status:    domain.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	roleRepo := &recordingRoleRepo{
		findByNameResult: nil, // FindByName always returns ErrRoleNotFound
	}

	svc := newAdminServiceWithRoleRepo(userRepo, roleRepo)

	_, err := svc.ReplaceUserRoles(
		context.Background(),
		targetUserID.String(),
		[]string{"nonexistent_role"},
		map[string][]string{},
		uuid.New().String(),
	)
	if err == nil {
		t.Fatal("expected an error for unknown role name, got nil")
	}

	if len(roleRepo.replaceCallArgs) != 0 {
		t.Errorf("expected no ReplaceUserRoles call on unknown role, got %d", len(roleRepo.replaceCallArgs))
	}
}

// TC-BF-12: ReplaceUserRoles with an invalid userID returns an error before
// any repository calls.
func TestReplaceUserRoles_InvalidUserID_ReturnsError(t *testing.T) {
	roleRepo := &recordingRoleRepo{}
	svc := newAdminServiceWithRoleRepo(newStubUserRepo(), roleRepo)

	_, err := svc.ReplaceUserRoles(
		context.Background(),
		"not-a-valid-uuid",
		[]string{"admin"},
		map[string][]string{},
		uuid.New().String(),
	)
	if err == nil {
		t.Fatal("expected an error for invalid user ID, got nil")
	}

	if len(roleRepo.replaceCallArgs) != 0 {
		t.Errorf("expected no ReplaceUserRoles call for invalid user ID, got %d", len(roleRepo.replaceCallArgs))
	}
}
