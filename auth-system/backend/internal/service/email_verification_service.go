// internal/service/email_verification_service.go
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

// EmailVerificationService handles email address confirmation flows.
type EmailVerificationService interface {
	VerifyEmail(ctx context.Context, rawToken string) error
	ResendVerification(ctx context.Context, email string) error
}

type emailVerificationServiceImpl struct {
	userRepo  domain.UserRepository
	tokenRepo domain.TokenRepository
	auditRepo domain.AuditRepository
	emailCh   chan<- EmailTask
	tokenTTL  time.Duration
}

// NewEmailVerificationService constructs an EmailVerificationService with all
// dependencies injected.
func NewEmailVerificationService(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	auditRepo domain.AuditRepository,
	emailCh chan<- EmailTask,
	tokenTTL time.Duration,
) EmailVerificationService {
	return &emailVerificationServiceImpl{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		auditRepo: auditRepo,
		emailCh:   emailCh,
		tokenTTL:  tokenTTL,
	}
}

// VerifyEmail hashes rawToken, looks up the verification record, activates the
// user, marks the token consumed, and writes an audit event.
func (s *emailVerificationServiceImpl) VerifyEmail(ctx context.Context, rawToken string) error {
	tokenHash := crypto.HashTokenString(rawToken)

	record, err := s.tokenRepo.FindEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("find verification token: %w", err)
	}

	if !record.IsValid() {
		return domain.ErrInvalidRefreshToken
	}

	now := time.Now()
	activeStatus := domain.UserStatusActive

	if _, err := s.userRepo.Update(ctx, record.UserID, domain.UpdateUserInput{
		Status:          &activeStatus,
		EmailVerifiedAt: &now,
	}); err != nil {
		return fmt.Errorf("activate user: %w", err)
	}

	if err := s.tokenRepo.MarkEmailVerificationTokenUsed(ctx, tokenHash); err != nil {
		return fmt.Errorf("mark verification token used: %w", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventEmailVerified,
		ActorID:      &record.UserID,
		TargetUserID: &record.UserID,
	})

	return nil
}

// ResendVerification generates a fresh verification token for an unverified
// user. It returns nil silently when the email is not found to prevent
// account enumeration.
func (s *emailVerificationServiceImpl) ResendVerification(ctx context.Context, email string) error {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.ErrInvalidEmail
	}

	user, err := s.userRepo.FindByEmail(ctx, normalized)
	if err != nil {
		// Anti-enumeration: do not reveal whether the address exists.
		return nil
	}

	if user.IsVerified() {
		return nil
	}

	rawToken, tokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}

	expiresAt := time.Now().Add(s.tokenTTL)
	if err := s.tokenRepo.CreateEmailVerificationToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store verification token: %w", err)
	}

	s.enqueueEmail(EmailTask{
		Type:      EmailTypeVerification,
		ToEmail:   user.Email,
		ToName:    user.FullName(),
		Token:     rawToken,
		ExpiresAt: expiresAt,
	})

	return nil
}

func (s *emailVerificationServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped", "type", task.Type, "to", task.ToEmail)
	}
}

func (s *emailVerificationServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}
