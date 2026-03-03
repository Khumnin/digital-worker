// internal/service/profile_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/pkg/crypto"
	"tigersoft/auth-system/pkg/validator"
)

// ProfileService handles self-service user profile operations.
type ProfileService interface {
	// GetProfile returns the full profile for the given user.
	GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResult, error)

	// UpdateProfile updates the user's first and/or last name.
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*ProfileResult, error)

	// ChangePassword verifies the current password, validates the new one against
	// the tenant policy, updates the hash, and revokes all active sessions and
	// outstanding OAuth authorization codes.
	ChangePassword(ctx context.Context, input ChangePasswordInput) error

	// RequestEmailChange validates the new address and queues a verification
	// email to the new address before any change is applied.
	RequestEmailChange(ctx context.Context, input EmailChangeInput) error

	// EraseOwnAccount performs a full GDPR self-erasure after verifying the
	// user's current password. This action is irreversible.
	EraseOwnAccount(ctx context.Context, userID uuid.UUID, password string) error
}

// ProfileResult is the public-facing representation of a user's profile.
type ProfileResult struct {
	UserID     uuid.UUID
	Email      string
	FirstName  string
	LastName   string
	MFAEnabled bool
	CreatedAt  time.Time
}

// UpdateProfileInput carries mutable display name fields.
type UpdateProfileInput struct {
	UserID    uuid.UUID
	FirstName *string
	LastName  *string
}

// ChangePasswordInput carries the fields for a self-service password change.
type ChangePasswordInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
	IPAddress       string
}

// EmailChangeInput carries the fields for requesting an email address change.
type EmailChangeInput struct {
	UserID   uuid.UUID
	NewEmail string
	TenantID string
}

// gdprProfileTombstoneUUID is the nil UUID tombstone for GDPR audit anonymization.
// Reuses the same value as adminServiceImpl for consistency.
var gdprProfileTombstoneUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type profileServiceImpl struct {
	userRepo          domain.UserRepository
	sessionRepo       domain.SessionRepository
	mfaRepo           domain.MFARepository
	socialAccountRepo domain.SocialAccountRepository
	codeRepo          domain.AuthorizationCodeRepository
	auditRepo         domain.AuditRepository
	emailCh           chan<- EmailTask
	tokenTTL          time.Duration
}

// NewProfileService constructs a ProfileService with all dependencies injected.
func NewProfileService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	mfaRepo domain.MFARepository,
	socialAccountRepo domain.SocialAccountRepository,
	codeRepo domain.AuthorizationCodeRepository,
	auditRepo domain.AuditRepository,
	emailCh chan<- EmailTask,
	tokenTTL time.Duration,
) ProfileService {
	return &profileServiceImpl{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		mfaRepo:           mfaRepo,
		socialAccountRepo: socialAccountRepo,
		codeRepo:          codeRepo,
		auditRepo:         auditRepo,
		emailCh:           emailCh,
		tokenTTL:          tokenTTL,
	}
}

// GetProfile fetches the user record and maps it to a ProfileResult.
func (s *profileServiceImpl) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user for profile: %w", err)
	}

	return userToProfile(user), nil
}

// UpdateProfile applies first_name and/or last_name changes to the user record.
// At least one field must be supplied; fields are validated as non-empty.
func (s *profileServiceImpl) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*ProfileResult, error) {
	if input.FirstName == nil && input.LastName == nil {
		return nil, fmt.Errorf("at least one of first_name or last_name must be provided")
	}

	if input.FirstName != nil && len(*input.FirstName) == 0 {
		return nil, fmt.Errorf("first_name must not be empty")
	}

	if input.LastName != nil && len(*input.LastName) == 0 {
		return nil, fmt.Errorf("last_name must not be empty")
	}

	user, err := s.userRepo.Update(ctx, input.UserID, domain.UpdateUserInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType:    domain.EventProfileUpdated,
		ActorID:      &input.UserID,
		TargetUserID: &input.UserID,
	})

	return userToProfile(user), nil
}

// ChangePassword verifies the current password, validates the new password
// against the tenant's policy, re-hashes, revokes all existing sessions, and
// invalidates any outstanding OAuth authorization codes issued to this user.
func (s *profileServiceImpl) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("find user for password change: %w", err)
	}

	if !crypto.VerifyPassword(input.CurrentPassword, user.PasswordHash) {
		return domain.ErrInvalidCredentials
	}

	// Validate new password against the default policy.
	defaultPolicy := domain.DefaultTenantConfig().PasswordPolicy
	if err := validator.CheckPasswordPolicy(input.NewPassword, defaultPolicy); err != nil {
		return err
	}

	newHash, err := crypto.HashPassword(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if _, err := s.userRepo.Update(ctx, input.UserID, domain.UpdateUserInput{
		PasswordHash: &newHash,
	}); err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}

	// Revoke all active sessions — a password change is a security event.
	revokedCount, err := s.sessionRepo.RevokeAllForUser(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to revoke sessions after password change",
			"user_id", input.UserID, "error", err)
	} else {
		slog.Info("sessions revoked after password change",
			"user_id", input.UserID, "count", revokedCount)
	}

	// Invalidate outstanding OAuth authorization codes — a changed password
	// means the user's prior authorization grants should not be exchangeable.
	if err := s.codeRepo.DeleteByUserID(ctx, input.UserID); err != nil {
		slog.Error("failed to delete OAuth codes after password change",
			"user_id", input.UserID, "error", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType:    domain.EventPasswordChanged,
		ActorID:      &input.UserID,
		ActorIP:      input.IPAddress,
		TargetUserID: &input.UserID,
	})

	return nil
}

// RequestEmailChange validates the new email address and sends a verification
// email to it. The actual email update does NOT happen here; it is applied only
// after the user clicks the verification link.
func (s *profileServiceImpl) RequestEmailChange(ctx context.Context, input EmailChangeInput) error {
	newEmail, err := domain.NormalizeEmail(input.NewEmail)
	if err != nil {
		return domain.ErrInvalidEmail
	}

	// Verify the new email is not already taken in this tenant.
	_, lookupErr := s.userRepo.FindByEmail(ctx, newEmail)
	if lookupErr == nil {
		return domain.ErrEmailAlreadyExists
	}

	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("find user for email change: %w", err)
	}

	s.enqueueEmail(EmailTask{
		Type:      EmailTypeVerification,
		ToEmail:   newEmail,
		ToName:    user.FullName(),
		ExpiresAt: time.Now().Add(s.tokenTTL),
		Extra: map[string]interface{}{
			"old_email": user.Email,
			"new_email": newEmail,
		},
	})

	return nil
}

// EraseOwnAccount performs a full GDPR self-erasure after the user confirms
// their current password. Steps mirror AdminService.EraseUser but are initiated
// by the account owner. This operation is irreversible.
func (s *profileServiceImpl) EraseOwnAccount(ctx context.Context, userID uuid.UUID, password string) error {
	// 1. Fetch user and verify password — confirmation gate for irreversible action.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user for self-erasure: %w", err)
	}

	if !crypto.VerifyPassword(password, user.PasswordHash) {
		return domain.ErrInvalidCredentials
	}

	// 2. Revoke all active sessions.
	if _, err := s.sessionRepo.RevokeAllForUser(ctx, userID); err != nil {
		slog.Error("self-erasure: failed to revoke sessions", "user_id", userID, "error", err)
	}

	// 3. Delete MFA backup codes.
	if err := s.mfaRepo.DeleteAllForUser(ctx, userID); err != nil {
		slog.Error("self-erasure: failed to delete MFA backup codes", "user_id", userID, "error", err)
	}

	// 4. Delete social account links.
	if err := s.socialAccountRepo.DeleteByUserID(ctx, userID); err != nil {
		slog.Error("self-erasure: failed to delete social accounts", "user_id", userID, "error", err)
	}

	// 5. Delete outstanding OAuth authorization codes.
	if err := s.codeRepo.DeleteByUserID(ctx, userID); err != nil {
		slog.Error("self-erasure: failed to delete OAuth codes", "user_id", userID, "error", err)
	}

	// 6. Anonymize PII in user record.
	if err := s.userRepo.AnonymizePII(ctx, userID); err != nil {
		return fmt.Errorf("anonymize user PII: %w", err)
	}

	// 7. Soft-delete the user record.
	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	// 8. Anonymize actor references in audit log.
	if err := s.auditRepo.AnonymizeActor(ctx, userID, gdprProfileTombstoneUUID); err != nil {
		slog.Error("self-erasure: failed to anonymize audit log actor", "user_id", userID, "error", err)
	}

	// 9. Write the erasure audit event.
	s.writeAudit(ctx, domain.AuditEvent{
		EventType:    domain.EventUserErased,
		TargetUserID: &userID,
		Metadata: map[string]interface{}{
			"erased":         true,
			"self_requested": true,
		},
	})

	return nil
}

func (s *profileServiceImpl) writeAudit(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write profile audit event", "event_type", event.EventType, "error", err)
	}
}

func (s *profileServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped", "type", task.Type, "to", task.ToEmail)
	}
}

func userToProfile(u *domain.User) *ProfileResult {
	return &ProfileResult{
		UserID:     u.ID,
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		MFAEnabled: u.MFAEnabled,
		CreatedAt:  u.CreatedAt,
	}
}
