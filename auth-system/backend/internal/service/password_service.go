// internal/service/password_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/pkg/crypto"
)

// PasswordService handles the forgot-password / reset-password flow.
type PasswordService interface {
	ForgotPassword(ctx context.Context, email string) error
	ValidateResetToken(ctx context.Context, rawToken string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}

type passwordServiceImpl struct {
	userRepo      domain.UserRepository
	sessionRepo   domain.SessionRepository
	tokenRepo     domain.TokenRepository
	auditRepo     domain.AuditRepository
	tenantRepo    domain.TenantRepository
	emailCh       chan<- EmailTask
	resetTokenTTL time.Duration
}

// NewPasswordService constructs a PasswordService with all dependencies
// injected.
func NewPasswordService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.TokenRepository,
	auditRepo domain.AuditRepository,
	tenantRepo domain.TenantRepository,
	emailCh chan<- EmailTask,
	resetTokenTTL time.Duration,
) PasswordService {
	return &passwordServiceImpl{
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		tokenRepo:     tokenRepo,
		auditRepo:     auditRepo,
		tenantRepo:    tenantRepo,
		emailCh:       emailCh,
		resetTokenTTL: resetTokenTTL,
	}
}

// ForgotPassword generates a password-reset token and enqueues the email.
// Returns nil even when the email is not registered to prevent enumeration.
func (s *passwordServiceImpl) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.ErrInvalidEmail
	}

	user, err := s.userRepo.FindByEmail(ctx, normalized)
	if err != nil {
		// Anti-enumeration: do not reveal whether the address exists.
		return nil
	}

	rawToken, tokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	expiresAt := time.Now().Add(s.resetTokenTTL)
	if err := s.tokenRepo.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	// Resolve the tenant display name for the email template.
	// A lookup failure here is non-fatal — fall back to the default.
	tenantName := ""
	if tenantID, ok := ctx.Value(pgdb.CtxKeyTenantID).(string); ok && tenantID != "" {
		if tenant, tenantErr := s.tenantRepo.FindBySlug(ctx, tenantID); tenantErr == nil {
			tenantName = tenant.Name
		} else {
			slog.Warn("forgot-password: could not resolve tenant name for email",
				"tenant_id", tenantID, "error", tenantErr)
		}
	}

	s.enqueueEmail(EmailTask{
		Type:       EmailTypePasswordReset,
		ToEmail:    user.Email,
		ToName:     user.FullName(),
		Token:      rawToken,
		ExpiresAt:  expiresAt,
		TenantName: tenantName,
	})

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventPasswordResetReq,
		ActorID:      &user.ID,
		TargetUserID: &user.ID,
	})

	return nil
}

// ValidateResetToken verifies that a raw password-reset token is still valid
// without consuming it. Callers use this to gate the "enter new password" step.
func (s *passwordServiceImpl) ValidateResetToken(ctx context.Context, rawToken string) error {
	tokenHash := crypto.HashTokenString(rawToken)

	record, err := s.tokenRepo.FindPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("find reset token: %w", err)
	}

	if !record.IsValid() {
		return domain.ErrInvalidRefreshToken
	}

	return nil
}

// ResetPassword validates the token, hashes the new password, updates the
// user record, revokes all active sessions, and writes an audit event.
func (s *passwordServiceImpl) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tokenHash := crypto.HashTokenString(rawToken)

	record, err := s.tokenRepo.FindPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("find reset token: %w", err)
	}

	if !record.IsValid() {
		return domain.ErrInvalidRefreshToken
	}

	passwordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if _, err := s.userRepo.Update(ctx, record.UserID, domain.UpdateUserInput{
		PasswordHash: &passwordHash,
	}); err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}

	if err := s.tokenRepo.MarkPasswordResetTokenUsed(ctx, tokenHash); err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}

	if _, err := s.sessionRepo.RevokeAllForUser(ctx, record.UserID); err != nil {
		// Log and continue — the password has already been changed, which is
		// the security-critical step. Failing to revoke sessions is recoverable.
		slog.Error("failed to revoke sessions after password reset",
			"user_id", record.UserID, "error", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventPasswordResetDone,
		ActorID:      &record.UserID,
		TargetUserID: &record.UserID,
	})

	return nil
}

func (s *passwordServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped", "type", task.Type, "to", task.ToEmail)
	}
}

func (s *passwordServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}
