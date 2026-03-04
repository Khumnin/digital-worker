// internal/service/session_service.go
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
	"tigersoft/auth-system/pkg/jwtutil"
)

// SessionService handles refresh-token rotation and access-token re-issuance.
type SessionService interface {
	Refresh(ctx context.Context, rawRefreshToken string, ipAddress, userAgent string) (*RefreshResult, error)
}

// RefreshResult carries the newly issued token pair returned from a token
// refresh operation.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type sessionServiceImpl struct {
	userRepo       domain.UserRepository
	sessionRepo    domain.SessionRepository
	auditRepo      domain.AuditRepository
	roleRepo       domain.RoleRepository
	keyStore       jwtutil.Signer
	accessTokenTTL time.Duration
}

// NewSessionService constructs a SessionService with all dependencies injected.
func NewSessionService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	auditRepo domain.AuditRepository,
	roleRepo domain.RoleRepository,
	keyStore jwtutil.Signer,
	accessTokenTTL time.Duration,
) SessionService {
	return &sessionServiceImpl{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		auditRepo:      auditRepo,
		roleRepo:       roleRepo,
		keyStore:       keyStore,
		accessTokenTTL: accessTokenTTL,
	}
}

// Refresh validates a refresh token, detects token reuse, rotates the token
// pair, and returns the new access + refresh tokens.
func (s *sessionServiceImpl) Refresh(
	ctx context.Context,
	rawRefreshToken string,
	ipAddress, userAgent string,
) (*RefreshResult, error) {
	tokenHash := crypto.HashTokenString(rawRefreshToken)

	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}

	// Expired session — caller must re-authenticate.
	if session.IsExpired() {
		return nil, domain.ErrSessionExpired
	}

	// Revoked session signals token reuse: revoke the entire family to
	// contain the breach and alert the security log.
	if session.IsRevoked {
		if _, revokeErr := s.sessionRepo.RevokeByFamilyID(ctx, session.FamilyID); revokeErr != nil {
			slog.Error("failed to revoke session family on token reuse",
				"family_id", session.FamilyID, "error", revokeErr)
		}

		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType:    domain.EventSuspiciousTokenReuse,
			ActorID:      &session.UserID,
			ActorIP:      ipAddress,
			ActorUA:      userAgent,
			TargetUserID: &session.UserID,
			Metadata: map[string]interface{}{
				"family_id": session.FamilyID.String(),
			},
		})

		return nil, domain.ErrSuspiciousTokenReuse
	}

	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !user.IsActive() {
		return nil, domain.ErrAccountDisabled
	}

	// Read the tenant slug from context (set by RequireTenant middleware).
	tenantSlug, _ := ctx.Value(pgdb.CtxKeyTenantID).(string)

	// Resolve the user's current roles for the new access token.
	systemRoles, moduleRoles := resolveRoles(ctx, s.roleRepo, user.ID)

	// Issue a new JWT access token.
	accessToken, err := s.keyStore.Sign(jwtutil.Claims{
		Subject:     user.ID.String(),
		Email:       user.Email,
		TenantID:    tenantSlug,
		Roles:       systemRoles,
		ModuleRoles: moduleRoles,
		TTL:         s.accessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Generate a new opaque refresh token.
	rawNewToken, newTokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Revoke the consumed token.
	if err := s.sessionRepo.RevokeByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}

	// Persist the new session in the same rotation family.
	newSession := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: newTokenHash,
		FamilyID:         session.FamilyID,
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		IssuedAt:         time.Now(),
		ExpiresAt:        session.ExpiresAt, // Preserve the original absolute expiry.
		LastUsedAt:       time.Now(),
		IsRevoked:        false,
	}

	if err := s.sessionRepo.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("create new session: %w", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventTokenRefreshed,
		ActorID:      &user.ID,
		ActorIP:      ipAddress,
		ActorUA:      userAgent,
		TargetUserID: &user.ID,
	})

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: rawNewToken,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *sessionServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}
