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
func (r *stubUserRepo) ListByTenant(_ context.Context, _, _ int) ([]*domain.User, int, error) {
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
func (r *stubRoleRepo) GetUserRoles(_ context.Context, _ uuid.UUID) ([]*domain.Role, error)     { return nil, nil }
func (r *stubRoleRepo) ReplaceUserRoles(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error    { return nil }

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

	user, err := svc.InviteUser(context.Background(), "new@example.com", "New", "User", "")

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

	user, err := svc.InviteUser(context.Background(), "pending@example.com", "Same", "Person", "")

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

	user, err := svc.InviteUser(context.Background(), "disabled@example.com", "Re", "Enable", "")

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

	_, err := svc.InviteUser(context.Background(), "active@example.com", "Active", "User", "")

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

	_, err := svc.InviteUser(context.Background(), "deleted@example.com", "Ghost", "User", "")

	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

// TC-6: Invalid email format returns ErrInvalidEmail.
func TestInviteUser_InvalidEmail_ReturnsInvalidEmail(t *testing.T) {
	svc := newTestAdminService(newStubUserRepo(), &stubTokenRepo{})

	_, err := svc.InviteUser(context.Background(), "not-an-email", "Bad", "Email", "")

	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got: %v", err)
	}
}
