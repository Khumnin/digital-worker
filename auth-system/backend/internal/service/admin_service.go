// internal/service/admin_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/pkg/crypto"
)

// gdprTombstoneUUID is the nil UUID used as a tombstone actor_id for GDPR-erased users.
// All audit log entries previously attributed to the erased user are updated to this value,
// preserving the audit trail shape while severing the link to the deleted identity.
var gdprTombstoneUUID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// AdminService exposes privileged user-management operations reserved for
// tenant administrators.
type AdminService interface {
	InviteUser(ctx context.Context, email, firstName, lastName string) (*domain.User, error)
	DisableUser(ctx context.Context, userID string) error
	// DeleteUser is kept for backward compatibility. It now delegates to EraseUser,
	// performing a full GDPR erasure rather than a simple soft delete.
	DeleteUser(ctx context.Context, userID string) error
	// EraseUser performs a full GDPR right-to-erasure sequence:
	// revoke sessions, delete MFA codes, delete social links, delete OAuth codes,
	// anonymize PII, soft-delete the user record, and anonymize audit log references.
	EraseUser(ctx context.Context, targetUserID string, requestedBy uuid.UUID) error
	ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
}

type adminServiceImpl struct {
	userRepo          domain.UserRepository
	sessionRepo       domain.SessionRepository
	roleRepo          domain.RoleRepository
	tokenRepo         domain.TokenRepository
	auditRepo         domain.AuditRepository
	mfaRepo           domain.MFARepository
	socialAccountRepo domain.SocialAccountRepository
	codeRepo          domain.AuthorizationCodeRepository
	emailCh           chan<- EmailTask
}

// NewAdminService constructs an AdminService with all dependencies injected.
func NewAdminService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	roleRepo domain.RoleRepository,
	auditRepo domain.AuditRepository,
	mfaRepo domain.MFARepository,
	socialAccountRepo domain.SocialAccountRepository,
	codeRepo domain.AuthorizationCodeRepository,
	emailCh chan<- EmailTask,
) AdminService {
	return &adminServiceImpl{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		roleRepo:          roleRepo,
		auditRepo:         auditRepo,
		mfaRepo:           mfaRepo,
		socialAccountRepo: socialAccountRepo,
		codeRepo:          codeRepo,
		emailCh:           emailCh,
	}
}

// WithTokenRepo sets the token repository on an already-constructed
// AdminService. Call this immediately after NewAdminService when InviteUser
// must persist verification tokens.
func WithTokenRepo(svc AdminService, tokenRepo domain.TokenRepository) AdminService {
	impl, ok := svc.(*adminServiceImpl)
	if !ok {
		return svc
	}
	impl.tokenRepo = tokenRepo
	return impl
}

// invitationTTL is the validity window for admin-issued invitation links.
const invitationTTL = 7 * 24 * time.Hour

// InviteUser creates an unverified user account, generates a verification
// token, and sends an invitation email.
func (s *adminServiceImpl) InviteUser(ctx context.Context, email, firstName, lastName string) (*domain.User, error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return nil, domain.ErrInvalidEmail
	}

	user, err := s.userRepo.Create(ctx, domain.CreateUserInput{
		Email:        normalized,
		PasswordHash: "",
		FirstName:    firstName,
		LastName:     lastName,
		Status:       domain.UserStatusUnverified,
	})
	if err != nil {
		return nil, fmt.Errorf("create invited user: %w", err)
	}

	rawToken, tokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate invitation token: %w", err)
	}

	expiresAt := time.Now().Add(invitationTTL)

	if s.tokenRepo != nil {
		if storeErr := s.tokenRepo.CreateEmailVerificationToken(ctx, user.ID, tokenHash, expiresAt); storeErr != nil {
			return nil, fmt.Errorf("store invitation token: %w", storeErr)
		}
	}

	s.enqueueEmail(EmailTask{
		Type:      EmailTypeInvitation,
		ToEmail:   user.Email,
		ToName:    user.FullName(),
		Token:     rawToken,
		ExpiresAt: expiresAt,
	})

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventUserInvited,
		TargetUserID: &user.ID,
		Metadata: map[string]interface{}{
			"email": user.Email,
		},
	})

	return user, nil
}

// DisableUser sets a user's status to Disabled and revokes all active sessions.
func (s *adminServiceImpl) DisableUser(ctx context.Context, userID string) error {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	disabled := domain.UserStatusDisabled

	if _, err := s.userRepo.Update(ctx, parsedID, domain.UpdateUserInput{
		Status: &disabled,
	}); err != nil {
		return fmt.Errorf("disable user: %w", err)
	}

	if _, err := s.sessionRepo.RevokeAllForUser(ctx, parsedID); err != nil {
		slog.Error("failed to revoke sessions after disabling user",
			"user_id", parsedID, "error", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventUserDisabled,
		ActorID:      &parsedID,
		TargetUserID: &parsedID,
	})

	return nil
}

// DeleteUser delegates to EraseUser for full GDPR compliance on admin delete.
// requestedBy is set to the nil UUID since the caller context is not threaded
// through this backward-compatibility shim.
func (s *adminServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	return s.EraseUser(ctx, userID, gdprTombstoneUUID)
}

// EraseUser performs the full GDPR right-to-erasure sequence for a user:
//  1. Confirm the user exists.
//  2. Revoke all active sessions.
//  3. Delete MFA backup codes.
//  4. Delete social account links.
//  5. Delete outstanding OAuth authorization codes.
//  6. Anonymize PII in the user record.
//  7. Soft-delete the user record.
//  8. Anonymize actor_id references in the audit log.
//  9. Write a USER_ERASED audit event.
func (s *adminServiceImpl) EraseUser(ctx context.Context, targetUserID string, requestedBy uuid.UUID) error {
	parsedID, err := uuid.Parse(targetUserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// 1. Confirm user exists before proceeding.
	if _, err := s.userRepo.FindByID(ctx, parsedID); err != nil {
		return fmt.Errorf("find user for erasure: %w", err)
	}

	// 2. Revoke all sessions — invalidates all active JWTs immediately.
	if _, err := s.sessionRepo.RevokeAllForUser(ctx, parsedID); err != nil {
		slog.Error("erasure: failed to revoke sessions", "user_id", parsedID, "error", err)
	}

	// 3. Delete MFA backup codes.
	if err := s.mfaRepo.DeleteAllForUser(ctx, parsedID); err != nil {
		slog.Error("erasure: failed to delete MFA backup codes", "user_id", parsedID, "error", err)
	}

	// 4. Delete social account links (e.g. Google OAuth connections).
	if err := s.socialAccountRepo.DeleteByUserID(ctx, parsedID); err != nil {
		slog.Error("erasure: failed to delete social accounts", "user_id", parsedID, "error", err)
	}

	// 5. Delete outstanding OAuth authorization codes.
	if err := s.codeRepo.DeleteByUserID(ctx, parsedID); err != nil {
		slog.Error("erasure: failed to delete OAuth codes", "user_id", parsedID, "error", err)
	}

	// 6. Anonymize PII fields — replaces email/name/password with tombstone values.
	if err := s.userRepo.AnonymizePII(ctx, parsedID); err != nil {
		return fmt.Errorf("anonymize user PII: %w", err)
	}

	// 7. Soft-delete the user record (sets deleted_at timestamp).
	if err := s.userRepo.SoftDelete(ctx, parsedID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	// 8. Anonymize actor references in audit log.
	if err := s.auditRepo.AnonymizeActor(ctx, parsedID, gdprTombstoneUUID); err != nil {
		slog.Error("erasure: failed to anonymize audit log actor", "user_id", parsedID, "error", err)
	}

	// 9. Write the erasure audit event.
	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventUserErased,
		TargetUserID: &parsedID,
		Metadata: map[string]interface{}{
			"erased":       true,
			"requested_by": requestedBy.String(),
		},
	})

	return nil
}

// ListUsers returns a paginated list of users in the current tenant schema
// along with the total count.
func (s *adminServiceImpl) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	users, total, err := s.userRepo.ListByTenant(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return users, total, nil
}

func (s *adminServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped", "type", task.Type, "to", task.ToEmail)
	}
}

func (s *adminServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}
