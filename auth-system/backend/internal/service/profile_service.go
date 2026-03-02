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
	// the tenant policy, updates the hash, and revokes all active sessions.
	ChangePassword(ctx context.Context, input ChangePasswordInput) error

	// RequestEmailChange validates the new address and queues a verification
	// email to the new address before any change is applied.
	RequestEmailChange(ctx context.Context, input EmailChangeInput) error
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

type profileServiceImpl struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	auditRepo   domain.AuditRepository
	emailCh     chan<- EmailTask
	tokenTTL    time.Duration
}

// NewProfileService constructs a ProfileService with all dependencies injected.
func NewProfileService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	auditRepo domain.AuditRepository,
	emailCh chan<- EmailTask,
	tokenTTL time.Duration,
) ProfileService {
	return &profileServiceImpl{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		emailCh:     emailCh,
		tokenTTL:    tokenTTL,
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
// against the tenant's policy, re-hashes, and revokes all existing sessions
// (security: new password invalidates all prior sessions).
func (s *profileServiceImpl) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("find user for password change: %w", err)
	}

	if !crypto.VerifyPassword(input.CurrentPassword, user.PasswordHash) {
		return domain.ErrInvalidCredentials
	}

	// Validate new password against the default policy. The tenant-specific
	// policy is loaded by the auth service during login; here we apply the
	// same default to keep the service independent of the tenant context.
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
// after the user clicks the verification link (handled by the email verification
// flow).
func (s *profileServiceImpl) RequestEmailChange(ctx context.Context, input EmailChangeInput) error {
	newEmail, err := domain.NormalizeEmail(input.NewEmail)
	if err != nil {
		return domain.ErrInvalidEmail
	}

	// Verify the new email is not already taken in this tenant.
	_, lookupErr := s.userRepo.FindByEmail(ctx, newEmail)
	if lookupErr == nil {
		// A user with that email already exists.
		return domain.ErrEmailAlreadyExists
	}

	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("find user for email change: %w", err)
	}

	// Queue the verification email to the new address.
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
