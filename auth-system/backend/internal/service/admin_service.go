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

// AdminService exposes privileged user-management operations reserved for
// tenant administrators.
type AdminService interface {
	InviteUser(ctx context.Context, email, firstName, lastName string) (*domain.User, error)
	DisableUser(ctx context.Context, userID string) error
	DeleteUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
}

type adminServiceImpl struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	roleRepo    domain.RoleRepository
	tokenRepo   domain.TokenRepository
	auditRepo   domain.AuditRepository
	emailCh     chan<- EmailTask
}

// NewAdminService constructs an AdminService with all dependencies injected.
// tokenRepo is required to persist invitation verification tokens.
func NewAdminService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	roleRepo domain.RoleRepository,
	auditRepo domain.AuditRepository,
	emailCh chan<- EmailTask,
) AdminService {
	return &adminServiceImpl{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		roleRepo:    roleRepo,
		auditRepo:   auditRepo,
		emailCh:     emailCh,
	}
}

// WithTokenRepo sets the token repository on an already-constructed
// AdminService. Call this immediately after NewAdminService when InviteUser
// must persist verification tokens. This keeps the primary constructor
// signature stable while allowing the optional dependency.
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
// token, and sends an invitation email. The user has no password until they
// complete the invitation flow.
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

	// Persist the hashed token when a token repository is wired in.
	// WithTokenRepo must be called during wiring if token persistence is required.
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

// DeleteUser soft-deletes a user record and revokes all active sessions.
func (s *adminServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if err := s.userRepo.SoftDelete(ctx, parsedID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	if _, err := s.sessionRepo.RevokeAllForUser(ctx, parsedID); err != nil {
		slog.Error("failed to revoke sessions after deleting user",
			"user_id", parsedID, "error", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventUserDeleted,
		ActorID:      &parsedID,
		TargetUserID: &parsedID,
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
