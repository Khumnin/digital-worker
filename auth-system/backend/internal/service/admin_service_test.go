// internal/service/admin_service_test.go
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
// Minimal stub repositories
// ---------------------------------------------------------------------------

type stubUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
	created []*domain.User
	updated []*domain.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
	}
}

func (r *stubUserRepo) seed(u *domain.User) {
	r.byEmail[u.Email] = u
	r.byID[u.ID] = u
}

func (r *stubUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *stubUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *stubUserRepo) Create(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
	u := &domain.User{
		ID:        uuid.New(),
		Email:     in.Email,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Status:    in.Status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.byEmail[u.Email] = u
	r.byID[u.ID] = u
	r.created = append(r.created, u)
	return u, nil
}

func (r *stubUserRepo) Update(_ context.Context, id uuid.UUID, in domain.UpdateUserInput) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	if in.Status != nil {
		u.Status = *in.Status
	}
	r.updated = append(r.updated, u)
	return u, nil
}

// Unused repository methods required by the interface.
func (r *stubUserRepo) IncrementFailedLoginCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *stubUserRepo) ResetFailedLoginCount(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubUserRepo) SetLockedUntil(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (r *stubUserRepo) ListByTenant(_ context.Context, _, _ int, _ string) ([]*domain.User, int, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) SoftDelete(_ context.Context, _ uuid.UUID) error    { return nil }
func (r *stubUserRepo) AnonymizePII(_ context.Context, _ uuid.UUID) error  { return nil }

// stubTokenRepo records stored tokens.
type stubTokenRepo struct {
	verificationTokens []string // stored token hashes
}

func (r *stubTokenRepo) CreateEmailVerificationToken(_ context.Context, _ uuid.UUID, tokenHash string, _ time.Time) error {
	r.verificationTokens = append(r.verificationTokens, tokenHash)
	return nil
}
func (r *stubTokenRepo) FindEmailVerificationToken(_ context.Context, _ string) (*domain.EmailVerificationToken, error) {
	return nil, nil
}
func (r *stubTokenRepo) MarkEmailVerificationTokenUsed(_ context.Context, _ string) error { return nil }
func (r *stubTokenRepo) CreatePasswordResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (r *stubTokenRepo) FindPasswordResetToken(_ context.Context, _ string) (*domain.PasswordResetToken, error) {
	return nil, nil
}
func (r *stubTokenRepo) MarkPasswordResetTokenUsed(_ context.Context, _ string) error { return nil }

// stubAuditRepo discards all events.
type stubAuditRepo struct{}

func (r *stubAuditRepo) Append(_ context.Context, _ *domain.AuditEvent) error { return nil }
func (r *stubAuditRepo) List(_ context.Context, _ domain.AuditFilter) ([]*domain.AuditEvent, int, error) {
	return nil, 0, nil
}
func (r *stubAuditRepo) MarkArchived(_ context.Context, _ []uuid.UUID) error { return nil }
func (r *stubAuditRepo) ListForArchive(_ context.Context, _ time.Time, _ int) ([]*domain.AuditEvent, error) {
	return nil, nil
}
func (r *stubAuditRepo) AnonymizeActor(_ context.Context, _, _ uuid.UUID) error { return nil }

// stubSessionRepo discards all operations.
type stubSessionRepo struct{}

func (r *stubSessionRepo) FindByTokenHash(_ context.Context, _ string) (*domain.Session, error) {
	return nil, nil
}
func (r *stubSessionRepo) Create(_ context.Context, _ *domain.Session) error        { return nil }
func (r *stubSessionRepo) RevokeByTokenHash(_ context.Context, _ string) error      { return nil }
func (r *stubSessionRepo) RevokeByFamilyID(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (r *stubSessionRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (r *stubSessionRepo) CountActiveForUser(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

// stubRoleRepo — minimal no-op.
type stubRoleRepo struct{}

func (r *stubRoleRepo) FindByID(_ context.Context, _ uuid.UUID) (*domain.Role, error) { return nil, nil }
func (r *stubRoleRepo) FindByName(_ context.Context, _ string) (*domain.Role, error)  { return nil, nil }
func (r *stubRoleRepo) ListAll(_ context.Context) ([]*domain.Role, error)              { return nil, nil }
func (r *stubRoleRepo) Create(_ context.Context, _, _ string, _ *string) (*domain.Role, error) {
	return nil, nil
}
func (r *stubRoleRepo) Delete(_ context.Context, _ uuid.UUID) error                             { return nil }
func (r *stubRoleRepo) IsAssignedToAnyUser(_ context.Context, _ uuid.UUID) (bool, error)        { return false, nil }
func (r *stubRoleRepo) AssignToUser(_ context.Context, _, _, _ uuid.UUID) error                 { return nil }
func (r *stubRoleRepo) UnassignFromUser(_ context.Context, _, _ uuid.UUID) error                { return nil }
func (r *stubRoleRepo) GetUserRoles(_ context.Context, _ uuid.UUID) ([]*domain.Role, error) {
	return nil, nil
}
func (r *stubRoleRepo) GetUserRolesBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*domain.Role, error) {
	return map[uuid.UUID][]*domain.Role{}, nil
}
func (r *stubRoleRepo) ReplaceUserRoles(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ uuid.UUID) error {
	return nil
}

// stubMFARepo — no-op.
type stubMFARepo struct{}

func (r *stubMFARepo) CreateBackupCodes(_ context.Context, _ uuid.UUID, _ []string) error { return nil }
func (r *stubMFARepo) ConsumeBackupCode(_ context.Context, _ uuid.UUID, _ string) error   { return nil }
func (r *stubMFARepo) DeleteAllForUser(_ context.Context, _ uuid.UUID) error               { return nil }

// stubSocialAccountRepo — no-op.
type stubSocialAccountRepo struct{}

func (r *stubSocialAccountRepo) FindByProvider(_ context.Context, _, _ string) (*domain.SocialAccount, error) {
	return nil, nil
}
func (r *stubSocialAccountRepo) Create(_ context.Context, _ *domain.SocialAccount) error { return nil }
func (r *stubSocialAccountRepo) UpdateTokens(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}
func (r *stubSocialAccountRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

// stubAuthCodeRepo — no-op.
type stubAuthCodeRepo struct{}

func (r *stubAuthCodeRepo) Create(_ context.Context, _ *domain.AuthorizationCode) error { return nil }
func (r *stubAuthCodeRepo) FindByCodeHash(_ context.Context, _ string) (*domain.AuthorizationCode, error) {
	return nil, nil
}
func (r *stubAuthCodeRepo) MarkUsed(_ context.Context, _ string) error             { return nil }
func (r *stubAuthCodeRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error    { return nil }

// ---------------------------------------------------------------------------
// Helper: construct a ready-to-use AdminService with stub dependencies.
// ---------------------------------------------------------------------------

func newTestAdminService(userRepo *stubUserRepo, tokenRepo *stubTokenRepo) service.AdminService {
	emailCh := make(chan service.EmailTask, 10)
	svc := service.NewAdminService(
		userRepo,
		&stubSessionRepo{},
		&stubRoleRepo{},
		&stubAuditRepo{},
		&stubMFARepo{},
		&stubSocialAccountRepo{},
		&stubAuthCodeRepo{},
		nil, // tenantRepo not needed in unit tests; resolveTenantName falls back to slug
		emailCh,
	)
	return service.WithTokenRepo(svc, tokenRepo)
}

// ---------------------------------------------------------------------------
// InviteUser tests
// ---------------------------------------------------------------------------

// TC-1: Brand-new email creates a user and sends an invitation.
func TestInviteUser_NewEmail_CreatesUserAndSendsEmail(t *testing.T) {
	userRepo := newStubUserRepo()
	tokenRepo := &stubTokenRepo{}
	svc := newTestAdminService(userRepo, tokenRepo)

	user, err := svc.InviteUser(context.Background(), "new@example.com", "New", "User", "", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("got email %q, want new@example.com", user.Email)
	}
	if user.Status != domain.UserStatusUnverified {
		t.Errorf("got status %q, want unverified", user.Status)
	}
	if len(userRepo.created) != 1 {
		t.Errorf("expected 1 created user, got %d", len(userRepo.created))
	}
	if len(tokenRepo.verificationTokens) != 1 {
		t.Errorf("expected 1 token stored, got %d", len(tokenRepo.verificationTokens))
	}
}

// TC-2: Re-inviting an unverified user does NOT create a new record —
// it reuses the existing user and sends a fresh invitation token.
func TestInviteUser_UnverifiedEmail_ResendsWithoutCreatingNewUser(t *testing.T) {
	userRepo := newStubUserRepo()
	existingID := uuid.New()
	userRepo.seed(&domain.User{
		ID:     existingID,
		Email:  "pending@example.com",
		Status: domain.UserStatusUnverified,
	})

	tokenRepo := &stubTokenRepo{}
	svc := newTestAdminService(userRepo, tokenRepo)

	user, err := svc.InviteUser(context.Background(), "pending@example.com", "Same", "Person", "", "")

	if err != nil {
		t.Fatalf("expected no error for re-invite of unverified user, got: %v", err)
	}
	if user.ID != existingID {
		t.Errorf("expected existing user ID %s, got %s", existingID, user.ID)
	}
	if len(userRepo.created) != 0 {
		t.Errorf("expected no new user created on re-invite, got %d", len(userRepo.created))
	}
	if len(tokenRepo.verificationTokens) != 1 {
		t.Errorf("expected 1 new token issued, got %d", len(tokenRepo.verificationTokens))
	}
}

// TC-3: Re-inviting a disabled (suspended) user re-enables the account
// (sets status → unverified) and sends a fresh invitation.
func TestInviteUser_DisabledEmail_ReEnablesAndSendsInvite(t *testing.T) {
	userRepo := newStubUserRepo()
	existingID := uuid.New()
	userRepo.seed(&domain.User{
		ID:     existingID,
		Email:  "disabled@example.com",
		Status: domain.UserStatusDisabled,
	})

	tokenRepo := &stubTokenRepo{}
	svc := newTestAdminService(userRepo, tokenRepo)

	user, err := svc.InviteUser(context.Background(), "disabled@example.com", "Re", "Enable", "", "")

	if err != nil {
		t.Fatalf("expected no error for re-invite of disabled user, got: %v", err)
	}
	if user.ID != existingID {
		t.Errorf("expected existing user ID %s, got %s", existingID, user.ID)
	}
	if user.Status != domain.UserStatusUnverified {
		t.Errorf("expected status unverified after re-invite, got %q", user.Status)
	}
	if len(userRepo.created) != 0 {
		t.Errorf("expected no new user created, got %d", len(userRepo.created))
	}
	if len(tokenRepo.verificationTokens) != 1 {
		t.Errorf("expected 1 token stored, got %d", len(tokenRepo.verificationTokens))
	}
}

// TC-4: Inviting an already-active email returns ErrEmailAlreadyExists.
func TestInviteUser_ActiveEmail_ReturnsEmailAlreadyExists(t *testing.T) {
	userRepo := newStubUserRepo()
	userRepo.seed(&domain.User{
		ID:     uuid.New(),
		Email:  "active@example.com",
		Status: domain.UserStatusActive,
	})

	svc := newTestAdminService(userRepo, &stubTokenRepo{})

	_, err := svc.InviteUser(context.Background(), "active@example.com", "Active", "User", "", "")

	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

// TC-5: Inviting a deleted (GDPR-erased) email returns ErrEmailAlreadyExists.
func TestInviteUser_DeletedEmail_ReturnsEmailAlreadyExists(t *testing.T) {
	userRepo := newStubUserRepo()
	userRepo.seed(&domain.User{
		ID:     uuid.New(),
		Email:  "deleted@example.com",
		Status: domain.UserStatusDeleted,
	})

	svc := newTestAdminService(userRepo, &stubTokenRepo{})

	_, err := svc.InviteUser(context.Background(), "deleted@example.com", "Ghost", "User", "", "")

	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

// TC-6: Invalid email format returns ErrInvalidEmail.
func TestInviteUser_InvalidEmail_ReturnsInvalidEmail(t *testing.T) {
	svc := newTestAdminService(newStubUserRepo(), &stubTokenRepo{})

	_, err := svc.InviteUser(context.Background(), "not-an-email", "Bad", "Email", "", "")

	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// stubRoleRepoWithData for ListAllAdmins tests
// ---------------------------------------------------------------------------

// stubRoleRepoWithData returns configurable roles for a given user.
type stubRoleRepoWithData struct {
	rolesByUserID map[uuid.UUID][]*domain.Role
}

func (r *stubRoleRepoWithData) FindByID(_ context.Context, _ uuid.UUID) (*domain.Role, error) {
	return nil, nil
}
func (r *stubRoleRepoWithData) FindByName(_ context.Context, name string) (*domain.Role, error) {
	return &domain.Role{ID: uuid.New(), Name: name}, nil
}
func (r *stubRoleRepoWithData) ListAll(_ context.Context) ([]*domain.Role, error)   { return nil, nil }
func (r *stubRoleRepoWithData) Create(_ context.Context, _, _ string, _ *string) (*domain.Role, error) {
	return nil, nil
}
func (r *stubRoleRepoWithData) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubRoleRepoWithData) IsAssignedToAnyUser(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *stubRoleRepoWithData) AssignToUser(_ context.Context, _, _, _ uuid.UUID) error  { return nil }
func (r *stubRoleRepoWithData) UnassignFromUser(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *stubRoleRepoWithData) GetUserRoles(_ context.Context, id uuid.UUID) ([]*domain.Role, error) {
	return r.rolesByUserID[id], nil
}
func (r *stubRoleRepoWithData) GetUserRolesBatch(_ context.Context, ids []uuid.UUID) (map[uuid.UUID][]*domain.Role, error) {
	result := make(map[uuid.UUID][]*domain.Role)
	for _, id := range ids {
		if roles, ok := r.rolesByUserID[id]; ok {
			result[id] = roles
		}
	}
	return result, nil
}
func (r *stubRoleRepoWithData) ReplaceUserRoles(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ uuid.UUID) error {
	return nil
}

// stubUserRepoWithData returns configurable users per schema context.
// The schema name is ignored — all users are returned for any schema call.
type stubUserRepoWithData struct {
	users []*domain.User
}

func (r *stubUserRepoWithData) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *stubUserRepoWithData) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (r *stubUserRepoWithData) Create(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
	return nil, nil
}
func (r *stubUserRepoWithData) Update(_ context.Context, _ uuid.UUID, _ domain.UpdateUserInput) (*domain.User, error) {
	return nil, nil
}
func (r *stubUserRepoWithData) IncrementFailedLoginCount(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *stubUserRepoWithData) ResetFailedLoginCount(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubUserRepoWithData) SetLockedUntil(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (r *stubUserRepoWithData) ListByTenant(_ context.Context, limit, offset int, status string) ([]*domain.User, int, error) {
	filtered := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		if status == "" || string(u.Status) == status {
			filtered = append(filtered, u)
		}
	}
	return filtered, len(filtered), nil
}
func (r *stubUserRepoWithData) SoftDelete(_ context.Context, _ uuid.UUID) error   { return nil }
func (r *stubUserRepoWithData) AnonymizePII(_ context.Context, _ uuid.UUID) error { return nil }

// newTestAdminServiceFull builds an AdminService with a real tenant repo and
// configurable role data — used for ListAllAdmins tests.
func newTestAdminServiceFull(userRepo domain.UserRepository, roleRepo domain.RoleRepository, tenantRepo domain.TenantRepository) service.AdminService {
	emailCh := make(chan service.EmailTask, 10)
	return service.NewAdminService(
		userRepo,
		&stubSessionRepo{},
		roleRepo,
		&stubAuditRepo{},
		&stubMFARepo{},
		&stubSocialAccountRepo{},
		&stubAuthCodeRepo{},
		tenantRepo,
		emailCh,
	)
}

// ---------------------------------------------------------------------------
// ListUsers status-filter tests (service layer)
// ---------------------------------------------------------------------------

// statusFilterUserRepo is a variant of stubUserRepoWithData that records the
// status argument passed to ListByTenant so tests can assert it was forwarded.
type statusFilterUserRepo struct {
	stubUserRepoWithData
	capturedStatus string
}

func (r *statusFilterUserRepo) ListByTenant(_ context.Context, limit, offset int, status string) ([]*domain.User, int, error) {
	r.capturedStatus = status
	return r.stubUserRepoWithData.ListByTenant(context.Background(), limit, offset, status)
}

// TC-7: ListUsers with status="unverified" forwards the filter to the repo and
// returns only matching users.
func TestListUsers_StatusFilter_ForwardedToRepo(t *testing.T) {
	pendingID := uuid.New()
	activeID := uuid.New()

	repo := &statusFilterUserRepo{
		stubUserRepoWithData: stubUserRepoWithData{
			users: []*domain.User{
				{ID: pendingID, Email: "pending@example.com", Status: domain.UserStatusUnverified},
				{ID: activeID, Email: "active@example.com", Status: domain.UserStatusActive},
			},
		},
	}

	emailCh := make(chan service.EmailTask, 10)
	svc := service.NewAdminService(
		repo,
		&stubSessionRepo{},
		&stubRoleRepo{},
		&stubAuditRepo{},
		&stubMFARepo{},
		&stubSocialAccountRepo{},
		&stubAuthCodeRepo{},
		nil,
		emailCh,
	)

	cases := []struct {
		name           string
		statusArg      string
		wantCount      int
		wantRepoStatus string
	}{
		{
			name:           "no filter returns all users",
			statusArg:      "",
			wantCount:      2,
			wantRepoStatus: "",
		},
		{
			name:           "unverified filter returns only pending users",
			statusArg:      "unverified",
			wantCount:      1,
			wantRepoStatus: "unverified",
		},
		{
			name:           "active filter returns only active users",
			statusArg:      "active",
			wantCount:      1,
			wantRepoStatus: "active",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo.capturedStatus = "" // reset between runs

			users, total, err := svc.ListUsers(context.Background(), 100, 0, tc.statusArg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tc.wantCount {
				t.Errorf("total: got %d, want %d", total, tc.wantCount)
			}
			if len(users) != tc.wantCount {
				t.Errorf("len(users): got %d, want %d", len(users), tc.wantCount)
			}
			if repo.capturedStatus != tc.wantRepoStatus {
				t.Errorf("repo received status=%q, want %q", repo.capturedStatus, tc.wantRepoStatus)
			}
		})
	}
}

// TC-8: ListUsers with status filter returns correct total (used by the
// pending-count banner on the frontend).
func TestListUsers_PendingCount_ReflectsOnlyPendingUsers(t *testing.T) {
	repo := &statusFilterUserRepo{
		stubUserRepoWithData: stubUserRepoWithData{
			users: []*domain.User{
				{ID: uuid.New(), Email: "p1@example.com", Status: domain.UserStatusUnverified},
				{ID: uuid.New(), Email: "a1@example.com", Status: domain.UserStatusActive},
				{ID: uuid.New(), Email: "a2@example.com", Status: domain.UserStatusActive},
				{ID: uuid.New(), Email: "d1@example.com", Status: domain.UserStatusDisabled},
			},
		},
	}

	emailCh := make(chan service.EmailTask, 10)
	svc := service.NewAdminService(
		repo,
		&stubSessionRepo{},
		&stubRoleRepo{},
		&stubAuditRepo{},
		&stubMFARepo{},
		&stubSocialAccountRepo{},
		&stubAuthCodeRepo{},
		nil,
		emailCh,
	)

	// Simulate the banner call: page_size=1, status="unverified".
	_, total, err := svc.ListUsers(context.Background(), 1, 0, "unverified")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only 1 user has status=unverified — total must be 1, not 4.
	if total != 1 {
		t.Errorf("pending count: got %d, want 1 — status filter not applied to total", total)
	}
}

// ---------------------------------------------------------------------------
// ListAllAdmins tests
// ---------------------------------------------------------------------------

// TC-9: Returns only admin/super_admin users across all active tenants.
func TestListAllAdmins_ReturnsOnlyAdminUsers(t *testing.T) {
	adminID := uuid.New()
	regularID := uuid.New()
	superAdminID := uuid.New()

	userRepo := &stubUserRepoWithData{
		users: []*domain.User{
			{ID: adminID, Email: "admin@example.com", Status: domain.UserStatusActive},
			{ID: regularID, Email: "user@example.com", Status: domain.UserStatusActive},
			{ID: superAdminID, Email: "super@example.com", Status: domain.UserStatusActive},
		},
	}

	roleRepo := &stubRoleRepoWithData{
		rolesByUserID: map[uuid.UUID][]*domain.Role{
			adminID:      {{ID: uuid.New(), Name: "admin"}},
			regularID:    {{ID: uuid.New(), Name: "user"}},
			superAdminID: {{ID: uuid.New(), Name: "super_admin"}},
		},
	}

	tenantRepo := newStubTenantRepo()
	tenantRepo.seed(&domain.Tenant{
		ID:         uuid.New(),
		Slug:       "acme",
		Name:       "ACME Corp",
		SchemaName: "tenant_acme",
		Status:     domain.TenantStatusActive,
	})

	svc := newTestAdminServiceFull(userRepo, roleRepo, tenantRepo)

	results, total, err := svc.ListAllAdmins(context.Background(), 100, 0, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2 (admin+super_admin), got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	for _, uwr := range results {
		if uwr.TenantID != "acme" {
			t.Errorf("expected TenantID=acme, got %q", uwr.TenantID)
		}
		if uwr.TenantName != "ACME Corp" {
			t.Errorf("expected TenantName=ACME Corp, got %q", uwr.TenantName)
		}
	}
}

// TC-8: Suspended tenants are skipped.
func TestListAllAdmins_SkipsSuspendedTenants(t *testing.T) {
	adminID := uuid.New()

	userRepo := &stubUserRepoWithData{
		users: []*domain.User{
			{ID: adminID, Email: "admin@example.com", Status: domain.UserStatusActive},
		},
	}
	roleRepo := &stubRoleRepoWithData{
		rolesByUserID: map[uuid.UUID][]*domain.Role{
			adminID: {{ID: uuid.New(), Name: "admin"}},
		},
	}

	tenantRepo := newStubTenantRepo()
	tenantRepo.seed(&domain.Tenant{
		ID:         uuid.New(),
		Slug:       "suspendedco",
		Name:       "Suspended Co",
		SchemaName: "tenant_suspendedco",
		Status:     domain.TenantStatusSuspended,
	})

	svc := newTestAdminServiceFull(userRepo, roleRepo, tenantRepo)

	results, total, err := svc.ListAllAdmins(context.Background(), 100, 0, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total=0 (suspended tenant skipped), got %d", total)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TC-9: In-memory pagination works correctly.
func TestListAllAdmins_PaginationApplied(t *testing.T) {
	ids := [3]uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	users := make([]*domain.User, 3)
	rolesByUser := make(map[uuid.UUID][]*domain.Role, 3)
	for i := 0; i < 3; i++ {
		users[i] = &domain.User{ID: ids[i], Email: "admin@example.com", Status: domain.UserStatusActive}
		rolesByUser[ids[i]] = []*domain.Role{{ID: uuid.New(), Name: "admin"}}
	}

	userRepo := &stubUserRepoWithData{users: users}
	roleRepo := &stubRoleRepoWithData{rolesByUserID: rolesByUser}

	tenantRepo := newStubTenantRepo()
	tenantRepo.seed(&domain.Tenant{
		ID:         uuid.New(),
		Slug:       "bigcorp",
		Name:       "Big Corp",
		SchemaName: "tenant_bigcorp",
		Status:     domain.TenantStatusActive,
	})

	svc := newTestAdminServiceFull(userRepo, roleRepo, tenantRepo)

	// Page 1: limit=2, offset=0
	page1, total, err := svc.ListAllAdmins(context.Background(), 2, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(page1))
	}

	// Page 2: limit=2, offset=2
	page2, _, err := svc.ListAllAdmins(context.Background(), 2, 2, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 result on page 2, got %d", len(page2))
	}
}
