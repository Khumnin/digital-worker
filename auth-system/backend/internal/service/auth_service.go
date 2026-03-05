// internal/service/auth_service.go
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
	"tigersoft/auth-system/pkg/validator"
)

// AuthService defines the interface for the core authentication use cases.
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterResult, error)
	Login(ctx context.Context, input LoginInput) (*LoginResult, error)
	Logout(ctx context.Context, input LogoutInput) error
	LogoutAll(ctx context.Context, input LogoutAllInput) (int, error)
}

type authServiceImpl struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	tokenRepo   domain.TokenRepository
	auditRepo   domain.AuditRepository
	tenantRepo  domain.TenantRepository
	roleRepo    domain.RoleRepository
	jwtSvc      jwtutil.Signer
	emailCh     chan<- EmailTask
	mfaSvc      MFAService
	cfg         AuthServiceConfig
}

// AuthServiceConfig holds configuration values needed by AuthService.
type AuthServiceConfig struct {
	AccessTokenTTL         time.Duration
	SessionDefaultTTL      time.Duration
	VerificationTokenTTL   time.Duration
	LockoutThreshold       int
	LockoutDurationSeconds int
}

// NewAuthService constructs an AuthService with all required dependencies injected.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.TokenRepository,
	auditRepo domain.AuditRepository,
	tenantRepo domain.TenantRepository,
	roleRepo domain.RoleRepository,
	jwtSvc jwtutil.Signer,
	emailCh chan<- EmailTask,
	mfaSvc MFAService,
	cfg AuthServiceConfig,
) AuthService {
	return &authServiceImpl{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenRepo:   tokenRepo,
		auditRepo:   auditRepo,
		tenantRepo:  tenantRepo,
		roleRepo:    roleRepo,
		jwtSvc:      jwtSvc,
		emailCh:     emailCh,
		mfaSvc:      mfaSvc,
		cfg:         cfg,
	}
}

type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	IPAddress string
	UserAgent string
}

type RegisterResult struct {
	UserID string
	Email  string
	Status string
}

type LoginInput struct {
	Email     string
	Password  string
	TOTPCode  string // optional; required only when the account has MFA enabled
	IPAddress string
	UserAgent string
}

type LoginResult struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int
	RefreshExpiresIn int
}

type LogoutInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type LogoutAllInput struct {
	UserID    string
	IPAddress string
}

// Register creates a new user account in the tenant schema.
func (s *authServiceImpl) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		return nil, domain.ErrInvalidEmail
	}

	tenant, err := s.tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := validator.CheckPasswordPolicy(input.Password, tenant.Config.PasswordPolicy); err != nil {
		return nil, err
	}

	passwordHash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.Create(ctx, domain.CreateUserInput{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Status:       domain.UserStatusUnverified,
	})
	if err != nil {
		return nil, err
	}

	rawToken, tokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}

	expiresAt := time.Now().Add(s.cfg.VerificationTokenTTL)
	if err := s.tokenRepo.CreateEmailVerificationToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("store verification token: %w", err)
	}

	s.enqueueEmail(EmailTask{
		Type:      EmailTypeVerification,
		ToEmail:   user.Email,
		ToName:    user.FullName(),
		Token:     rawToken,
		ExpiresAt: expiresAt,
	})

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventUserRegistered,
		ActorID:      &user.ID,
		ActorIP:      input.IPAddress,
		ActorUA:      input.UserAgent,
		TargetUserID: &user.ID,
		Metadata: map[string]interface{}{
			"email": user.Email,
		},
	})

	slog.Info("user registered", "user_id", user.ID, "tenant_id", tenant.ID)

	return &RegisterResult{
		UserID: user.ID.String(),
		Email:  user.Email,
		Status: string(user.Status),
	}, nil
}

// Login authenticates a user and issues JWT + opaque refresh token.
func (s *authServiceImpl) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	tenant, err := s.tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType: domain.EventLoginFailure,
			ActorIP:   input.IPAddress,
			ActorUA:   input.UserAgent,
			Metadata: map[string]interface{}{
				"reason":          "user_not_found",
				"email_attempted": input.Email,
			},
		})
		return nil, domain.ErrInvalidCredentials
	}

	if user.IsLocked() {
		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType:    domain.EventLoginFailure,
			ActorID:      &user.ID,
			ActorIP:      input.IPAddress,
			ActorUA:      input.UserAgent,
			TargetUserID: &user.ID,
			Metadata: map[string]interface{}{
				"reason":          "account_locked",
				"email_attempted": input.Email,
			},
		})
		return nil, domain.ErrAccountLocked
	}

	if user.Status == domain.UserStatusDisabled {
		return nil, domain.ErrAccountDisabled
	}
	if user.Status == domain.UserStatusDeleted {
		return nil, domain.ErrInvalidCredentials
	}
	if user.Status == domain.UserStatusUnverified {
		return nil, domain.ErrEmailNotVerified
	}

	if !crypto.VerifyPassword(input.Password, user.PasswordHash) {
		newCount, incrErr := s.userRepo.IncrementFailedLoginCount(ctx, user.ID)
		if incrErr != nil {
			slog.Error("failed to increment failed login count", "user_id", user.ID, "error", incrErr)
		}

		threshold := s.cfg.LockoutThreshold
		if tenant.Config.LockoutThreshold > 0 {
			threshold = tenant.Config.LockoutThreshold
		}

		if newCount >= threshold {
			lockDuration := time.Duration(s.cfg.LockoutDurationSeconds) * time.Second
			if tenant.Config.LockoutDurationSeconds > 0 {
				lockDuration = time.Duration(tenant.Config.LockoutDurationSeconds) * time.Second
			}
			lockUntil := time.Now().Add(lockDuration)
			if lockErr := s.userRepo.SetLockedUntil(ctx, user.ID, lockUntil); lockErr != nil {
				slog.Error("failed to lock account", "user_id", user.ID, "error", lockErr)
			}

			s.writeAuditEvent(ctx, domain.AuditEvent{
				EventType:    domain.EventAccountLocked,
				ActorIP:      input.IPAddress,
				ActorUA:      input.UserAgent,
				TargetUserID: &user.ID,
				Metadata: map[string]interface{}{
					"locked_until": lockUntil,
					"email":        user.Email,
					"failed_count": newCount,
				},
			})
		}

		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType:    domain.EventLoginFailure,
			ActorID:      &user.ID,
			ActorIP:      input.IPAddress,
			ActorUA:      input.UserAgent,
			TargetUserID: &user.ID,
			Metadata: map[string]interface{}{
				"reason":          "invalid_password",
				"failed_count":    newCount,
				"email_attempted": input.Email,
			},
		})

		return nil, domain.ErrInvalidCredentials
	}

	if resetErr := s.userRepo.ResetFailedLoginCount(ctx, user.ID); resetErr != nil {
		slog.Error("failed to reset login count", "user_id", user.ID, "error", resetErr)
	}

	// MFA enforcement: if the tenant requires MFA but the user has not enrolled,
	// block login and prompt them to set up TOTP before proceeding.
	if tenant.Config.MFARequired && !user.MFAEnabled {
		return nil, domain.ErrMFAEnrollmentRequired
	}

	// MFA step-up: if the user has MFA enabled, a TOTP code is required.
	if user.MFAEnabled {
		if input.TOTPCode == "" {
			// Signal to the handler that MFA input is needed (not a hard failure).
			return nil, domain.ErrMFARequired
		}
		if err := s.mfaSvc.VerifyTOTP(ctx, user.ID, input.TOTPCode); err != nil {
			s.writeAuditEvent(ctx, domain.AuditEvent{
				EventType:    domain.EventLoginFailure,
				ActorID:      &user.ID,
				ActorIP:      input.IPAddress,
				ActorUA:      input.UserAgent,
				TargetUserID: &user.ID,
				Metadata: map[string]interface{}{
					"reason":          "invalid_totp",
					"email_attempted": input.Email,
				},
			})
			return nil, domain.ErrInvalidTOTPCode
		}
	}

	if _, updateErr := s.userRepo.Update(ctx, user.ID, domain.UpdateUserInput{
		LastLoginAt: timePtr(time.Now()),
	}); updateErr != nil {
		slog.Error("failed to update last_login_at", "user_id", user.ID, "error", updateErr)
	}

	sessionTTL := s.cfg.SessionDefaultTTL
	if tenant.Config.SessionTTLSeconds > 0 {
		sessionTTL = time.Duration(tenant.Config.SessionTTLSeconds) * time.Second
	}

	systemRoleNames, moduleRolesMap := s.resolveUserRoles(ctx, user.ID)

	accessToken, err := s.jwtSvc.Sign(jwtutil.Claims{
		Subject:     user.ID.String(),
		Email:       user.Email,
		TenantID:    tenant.Slug,
		Roles:       systemRoleNames,
		ModuleRoles: moduleRolesMap,
		TTL:         s.cfg.AccessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefreshToken, refreshTokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	familyID := uuid.New()
	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		FamilyID:         familyID,
		IPAddress:        input.IPAddress,
		UserAgent:        input.UserAgent,
		IssuedAt:         time.Now(),
		ExpiresAt:        time.Now().Add(sessionTTL),
		LastUsedAt:       time.Now(),
		IsRevoked:        false,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	method := "password"
	if input.TOTPCode != "" {
		method = "totp_verified"
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventLoginSuccess,
		ActorID:      &user.ID,
		ActorIP:      input.IPAddress,
		ActorUA:      input.UserAgent,
		TargetUserID: &user.ID,
		Metadata: map[string]interface{}{
			"email":    user.Email,
			"method":   method,
			"mfa_used": user.MFAEnabled,
		},
	})

	return &LoginResult{
		AccessToken:      accessToken,
		RefreshToken:     rawRefreshToken,
		ExpiresIn:        int(s.cfg.AccessTokenTTL.Seconds()),
		RefreshExpiresIn: int(sessionTTL.Seconds()),
	}, nil
}

// Logout revokes the specific refresh token presented.
func (s *authServiceImpl) Logout(ctx context.Context, input LogoutInput) error {
	_, tokenHash := crypto.HashToken(input.RefreshToken)

	// Fetch session before revoking to capture the actor identity for the audit trail.
	session, _ := s.sessionRepo.FindByTokenHash(ctx, tokenHash)

	if err := s.sessionRepo.RevokeByTokenHash(ctx, tokenHash); err != nil {
		slog.Warn("logout: session not found (idempotent)", "error", err)
	}

	event := domain.AuditEvent{
		EventType: domain.EventLogout,
		ActorIP:   input.IPAddress,
		ActorUA:   input.UserAgent,
	}
	if session != nil {
		event.ActorID = &session.UserID
		event.TargetUserID = &session.UserID
	}

	s.writeAuditEvent(ctx, event)

	return nil
}

// LogoutAll revokes all active refresh tokens for the authenticated user.
func (s *authServiceImpl) LogoutAll(ctx context.Context, input LogoutAllInput) (int, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}

	count, err := s.sessionRepo.RevokeAllForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke all sessions: %w", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventLogoutAll,
		ActorID:      &userID,
		ActorIP:      input.IPAddress,
		TargetUserID: &userID,
		Metadata:     map[string]interface{}{"sessions_revoked": count},
	})

	return count, nil
}

func (s *authServiceImpl) tenantFromContext(ctx context.Context) (*domain.Tenant, error) {
	tenantID, ok := ctx.Value(pgdb.CtxKeyTenantID).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant ID not in context")
	}
	return s.tenantRepo.FindBySlug(ctx, tenantID)
}

func (s *authServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped", "type", task.Type, "to", task.ToEmail)
	}
}

func (s *authServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}

// resolveUserRoles calls the shared resolveRoles helper on the auth service's roleRepo.
func (s *authServiceImpl) resolveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, map[string][]string) {
	return resolveRoles(ctx, s.roleRepo, userID)
}

func timePtr(t time.Time) *time.Time { return &t }

// EmailTask is a task submitted to the async email worker.
type EmailTask struct {
	Type       EmailType
	ToEmail    string
	ToName     string
	Token      string
	ExpiresAt  time.Time
	TenantSlug string // used to build tenant-scoped accept-invite / verify URLs
	Extra      map[string]interface{}
}

type EmailType string

const (
	EmailTypeVerification  EmailType = "verification"
	EmailTypePasswordReset EmailType = "password_reset"
	EmailTypeInvitation    EmailType = "invitation"
)
