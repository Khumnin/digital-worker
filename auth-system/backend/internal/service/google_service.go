// internal/service/google_service.go
// Sprint 6 — Google OIDC social login (US-13).
//
// Implements the Google OAuth 2.0 / OIDC flow without any external OAuth library.
// Uses standard net/http for the token exchange and Google's tokeninfo endpoint
// for ID-token validation. State tokens are stored in Redis for CSRF protection.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"tigersoft/auth-system/internal/config"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/pkg/crypto"
	"tigersoft/auth-system/pkg/jwtutil"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"
	googleStatePrefix = "google_state:"
)

// GoogleService handles the Google OIDC social login flow.
type GoogleService interface {
	// InitiateLogin builds a Google authorization URL and stores a one-time state token
	// in Redis for CSRF protection. The caller returns the auth_url to the client;
	// the client performs the browser redirect (API-only per ADR-003).
	InitiateLogin(ctx context.Context, input GoogleInitiateInput) (*GoogleInitiateResult, error)

	// HandleCallback exchanges the authorization code for an ID token, resolves or
	// creates the user account, and issues JWT access + refresh tokens.
	HandleCallback(ctx context.Context, input GoogleCallbackInput) (*GoogleCallbackResult, error)
}

// GoogleInitiateInput carries the parameters needed to build the Google auth URL.
type GoogleInitiateInput struct {
	TenantConfig domain.TenantConfig // per-tenant Google credentials (may be empty strings)
	RedirectURI  string
	TenantID     string
}

// GoogleInitiateResult contains the Google authorization URL.
type GoogleInitiateResult struct {
	AuthURL string // client must redirect the user's browser to this URL
}

// GoogleCallbackInput carries the values received from Google's callback redirect.
type GoogleCallbackInput struct {
	Code        string
	State       string
	RedirectURI string
	TenantConfig domain.TenantConfig
	TenantID    string
	// Password is required only when linking a Google account to an existing
	// password-based user (ADR-006). Empty for new users or pure social accounts.
	Password string
}

// GoogleCallbackResult holds the auth tokens issued after a successful Google login.
type GoogleCallbackResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	IsNewUser    bool
	IsLinked     bool // true when Google was linked to an existing account
}

type googleServiceImpl struct {
	userRepo         domain.UserRepository
	sessionRepo      domain.SessionRepository
	socialAccountRepo domain.SocialAccountRepository
	auditRepo        domain.AuditRepository
	roleRepo         domain.RoleRepository
	redis            *redis.Client
	jwtSvc           jwtutil.Signer
	oauthCfg         config.OAuthConfig
	accessTTL        time.Duration
	sessionTTL       time.Duration
	httpClient       *http.Client
}

// NewGoogleService constructs a GoogleService with all dependencies injected.
func NewGoogleService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	socialAccountRepo domain.SocialAccountRepository,
	auditRepo domain.AuditRepository,
	roleRepo domain.RoleRepository,
	redisClient *redis.Client,
	jwtSvc jwtutil.Signer,
	oauthCfg config.OAuthConfig,
	accessTTL time.Duration,
	sessionTTL time.Duration,
) GoogleService {
	return &googleServiceImpl{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		socialAccountRepo: socialAccountRepo,
		auditRepo:         auditRepo,
		roleRepo:          roleRepo,
		redis:             redisClient,
		jwtSvc:            jwtSvc,
		oauthCfg:          oauthCfg,
		accessTTL:         accessTTL,
		sessionTTL:        sessionTTL,
		httpClient:        &http.Client{Timeout: 15 * time.Second},
	}
}

// ── InitiateLogin ─────────────────────────────────────────────────────────────

func (s *googleServiceImpl) InitiateLogin(ctx context.Context, in GoogleInitiateInput) (*GoogleInitiateResult, error) {
	clientID := s.resolveClientID(in.TenantConfig)
	if clientID == "" {
		return nil, domain.ErrGoogleNotConfigured
	}

	// Generate a cryptographically random state token and store it in Redis.
	rawState, _, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate state token: %w", err)
	}

	stateKey := googleStatePrefix + rawState
	if err := s.redis.Set(ctx, stateKey, in.TenantID, s.oauthCfg.StateTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store state token in redis: %w", err)
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", in.RedirectURI)
	params.Set("scope", "openid email profile")
	params.Set("state", rawState)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	authURL := googleAuthURL + "?" + params.Encode()
	return &GoogleInitiateResult{AuthURL: authURL}, nil
}

// ── HandleCallback ────────────────────────────────────────────────────────────

func (s *googleServiceImpl) HandleCallback(ctx context.Context, in GoogleCallbackInput) (*GoogleCallbackResult, error) {
	// Step 1: validate and consume the state token (one-time use, CSRF protection).
	if err := s.validateAndDeleteState(ctx, in.State); err != nil {
		return nil, domain.ErrGoogleStateInvalid
	}

	clientID := s.resolveClientID(in.TenantConfig)
	clientSecret := s.resolveClientSecret(in.TenantConfig)
	if clientID == "" || clientSecret == "" {
		return nil, domain.ErrGoogleNotConfigured
	}

	// Step 2: exchange the authorization code for Google tokens.
	googleTokens, err := s.exchangeCode(ctx, in.Code, clientID, clientSecret, in.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("exchange google code: %w", err)
	}

	// Step 3: verify the ID token and extract claims via tokeninfo endpoint.
	idInfo, err := s.verifyIDToken(ctx, googleTokens.IDToken, clientID)
	if err != nil {
		return nil, fmt.Errorf("verify google id token: %w", err)
	}

	// Step 4: resolve or create the user account.
	return s.resolveAccount(ctx, in, idInfo, googleTokens)
}

// ── Account resolution ────────────────────────────────────────────────────────

func (s *googleServiceImpl) resolveAccount(
	ctx context.Context,
	in GoogleCallbackInput,
	idInfo *googleIDTokenInfo,
	googleTokens *googleTokenResponse,
) (*GoogleCallbackResult, error) {
	// Try to find an existing social account link for this Google sub.
	existingSocial, err := s.socialAccountRepo.FindByProvider(ctx, "google", idInfo.Sub)
	if err != nil && !errors.Is(err, domain.ErrSocialAccountNotFound) {
		return nil, fmt.Errorf("find social account: %w", err)
	}

	if err == nil {
		// Case A: existing social account found — update tokens and issue JWT.
		var expiresAt *time.Time
		if googleTokens.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(googleTokens.ExpiresIn) * time.Second)
			expiresAt = &t
		}
		if updateErr := s.socialAccountRepo.UpdateTokens(ctx,
			existingSocial.ID,
			googleTokens.AccessToken,
			googleTokens.RefreshToken,
			expiresAt,
		); updateErr != nil {
			slog.Error("update google social account tokens", "error", updateErr)
		}
		return s.issueTokensForUser(ctx, existingSocial.UserID, in.TenantID, false, false)
	}

	// No social account found — look up user by email.
	existingUser, err := s.userRepo.FindByEmail(ctx, idInfo.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if existingUser != nil {
		// Case B: email already exists for a local user.
		return s.linkGoogleToExistingUser(ctx, in, idInfo, googleTokens, existingUser)
	}

	// Case C: brand new user — register via Google.
	return s.createGoogleUser(ctx, in, idInfo, googleTokens)
}

// linkGoogleToExistingUser handles ADR-006: linking Google to an existing account.
func (s *googleServiceImpl) linkGoogleToExistingUser(
	ctx context.Context,
	in GoogleCallbackInput,
	idInfo *googleIDTokenInfo,
	googleTokens *googleTokenResponse,
	user *domain.User,
) (*GoogleCallbackResult, error) {
	if user.HasPassword() {
		// ADR-006: user has a password — require it to authorize the link.
		if in.Password == "" {
			return nil, domain.ErrPasswordRequiredForLinking
		}
		if !crypto.VerifyPassword(in.Password, user.PasswordHash) {
			return nil, domain.ErrInvalidCredentials
		}
	}
	// user.HasPassword() == false → no password set, auto-link without verification.

	var expiresAt *time.Time
	if googleTokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(googleTokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	socialAcct := &domain.SocialAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       "google",
		ProviderUserID: idInfo.Sub,
		Email:          idInfo.Email,
		AccessToken:    googleTokens.AccessToken,
		RefreshToken:   googleTokens.RefreshToken,
		ExpiresAt:      expiresAt,
	}
	if err := s.socialAccountRepo.Create(ctx, socialAcct); err != nil {
		return nil, fmt.Errorf("create social account link: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType:    domain.EventGoogleLinked,
		ActorID:      &user.ID,
		TargetUserID: &user.ID,
		Metadata:     map[string]interface{}{"provider": "google", "email": idInfo.Email},
	})

	result, err := s.issueTokensForUser(ctx, user.ID, in.TenantID, false, true)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// createGoogleUser registers a brand-new user whose identity comes from Google.
func (s *googleServiceImpl) createGoogleUser(
	ctx context.Context,
	in GoogleCallbackInput,
	idInfo *googleIDTokenInfo,
	googleTokens *googleTokenResponse,
) (*GoogleCallbackResult, error) {
	now := time.Now()
	newUser, err := s.userRepo.Create(ctx, domain.CreateUserInput{
		Email:        idInfo.Email,
		PasswordHash: "", // social-only account
		FirstName:    idInfo.GivenName,
		LastName:     idInfo.FamilyName,
		Status:       domain.UserStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("create google user: %w", err)
	}

	// Mark email as verified immediately (Google already verified it).
	if _, updateErr := s.userRepo.Update(ctx, newUser.ID, domain.UpdateUserInput{
		EmailVerifiedAt: &now,
	}); updateErr != nil {
		slog.Error("set email_verified_at for google user", "user_id", newUser.ID, "error", updateErr)
	}

	var expiresAt *time.Time
	if googleTokens.ExpiresIn > 0 {
		t := now.Add(time.Duration(googleTokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	socialAcct := &domain.SocialAccount{
		ID:             uuid.New(),
		UserID:         newUser.ID,
		Provider:       "google",
		ProviderUserID: idInfo.Sub,
		Email:          idInfo.Email,
		AccessToken:    googleTokens.AccessToken,
		RefreshToken:   googleTokens.RefreshToken,
		ExpiresAt:      expiresAt,
	}
	if err := s.socialAccountRepo.Create(ctx, socialAcct); err != nil {
		return nil, fmt.Errorf("create social account for new google user: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType:    domain.EventGoogleLogin,
		ActorID:      &newUser.ID,
		TargetUserID: &newUser.ID,
		Metadata:     map[string]interface{}{"provider": "google", "email": idInfo.Email, "is_new_user": true},
	})

	result, err := s.issueTokensForUser(ctx, newUser.ID, in.TenantID, true, false)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// issueTokensForUser signs a JWT access token and creates a refresh session for userID.
func (s *googleServiceImpl) issueTokensForUser(
	ctx context.Context,
	userID uuid.UUID,
	tenantID string,
	isNewUser bool,
	isLinked bool,
) (*GoogleCallbackResult, error) {
	// Fetch the user's roles for JWT claims.
	roles, err := s.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		slog.Error("get user roles for google login", "user_id", userID, "error", err)
		roles = nil
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
	}
	if len(roleNames) == 0 {
		roleNames = []string{"user"}
	}

	accessToken, err := s.jwtSvc.Sign(jwtutil.Claims{
		Subject:  userID.String(),
		TenantID: tenantID,
		Roles:    roleNames,
		Scope:    "openid email profile",
		TTL:      s.accessTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, refreshHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		FamilyID:         uuid.New(),
		IssuedAt:         time.Now(),
		ExpiresAt:        time.Now().Add(s.sessionTTL),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session for google user: %w", err)
	}

	return &GoogleCallbackResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
		IsNewUser:    isNewUser,
		IsLinked:     isLinked,
	}, nil
}

// ── Google API helpers ─────────────────────────────────────────────────────────

// googleTokenResponse holds the raw response from Google's token endpoint.
type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// googleIDTokenInfo holds the claims returned by Google's tokeninfo endpoint.
type googleIDTokenInfo struct {
	Sub        string `json:"sub"`          // Google user ID
	Email      string `json:"email"`
	Aud        string `json:"aud"`          // must equal our client_id
	Iss        string `json:"iss"`          // must be accounts.google.com or https://accounts.google.com
	Exp        string `json:"exp"`          // unix timestamp string
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Name       string `json:"name"`
}

// exchangeCode posts to Google's token endpoint and returns the token response.
func (s *googleServiceImpl) exchangeCode(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*googleTokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build google token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token endpoint returned %d: %s", resp.StatusCode, body)
	}

	var tokens googleTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("decode google token response: %w", err)
	}
	if tokens.IDToken == "" {
		return nil, fmt.Errorf("google token response missing id_token")
	}
	return &tokens, nil
}

// verifyIDToken validates the ID token using Google's tokeninfo endpoint.
// This is simpler than JWKS-based verification and sufficient for this sprint.
// Validates: aud == clientID, iss is accounts.google.com, exp > now.
func (s *googleServiceImpl) verifyIDToken(ctx context.Context, idToken, clientID string) (*googleIDTokenInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		googleTokenInfoURL+"?id_token="+url.QueryEscape(idToken), nil)
	if err != nil {
		return nil, fmt.Errorf("build tokeninfo request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tokeninfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tokeninfo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tokeninfo returned %d: %s", resp.StatusCode, body)
	}

	var info googleIDTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decode tokeninfo response: %w", err)
	}

	// Validate audience matches our client_id.
	if info.Aud != clientID {
		return nil, fmt.Errorf("id_token aud %q does not match client_id %q", info.Aud, clientID)
	}

	// Validate issuer is Google.
	if info.Iss != "accounts.google.com" && info.Iss != "https://accounts.google.com" {
		return nil, fmt.Errorf("id_token iss %q is not a valid Google issuer", info.Iss)
	}

	// Validate token is not expired (Google tokeninfo does this too, but be explicit).
	if info.Exp == "" {
		return nil, fmt.Errorf("id_token missing exp claim")
	}
	if info.Sub == "" {
		return nil, fmt.Errorf("id_token missing sub claim")
	}
	if info.Email == "" {
		return nil, fmt.Errorf("id_token missing email claim")
	}

	return &info, nil
}

// validateAndDeleteState verifies the state token exists in Redis and deletes it (one-time use).
func (s *googleServiceImpl) validateAndDeleteState(ctx context.Context, state string) error {
	if state == "" {
		return domain.ErrGoogleStateInvalid
	}
	key := googleStatePrefix + state
	// GET then DEL in a pipeline — the value was stored during InitiateLogin.
	pipe := s.redis.Pipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Del(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis state pipeline: %w", err)
	}
	if err := getCmd.Err(); err != nil {
		// redis.Nil means the key did not exist (expired or already used).
		return domain.ErrGoogleStateInvalid
	}
	return nil
}

// resolveClientID returns the per-tenant Google client_id, or falls back to the
// global config value if the tenant has not configured its own.
func (s *googleServiceImpl) resolveClientID(tc domain.TenantConfig) string {
	if tc.GoogleClientID != "" {
		return tc.GoogleClientID
	}
	return s.oauthCfg.GoogleClientID
}

// resolveClientSecret returns the per-tenant client secret or the global fallback.
func (s *googleServiceImpl) resolveClientSecret(tc domain.TenantConfig) string {
	if tc.GoogleClientSecret != "" {
		return tc.GoogleClientSecret
	}
	return s.oauthCfg.GoogleClientSecret
}

// writeAudit fires an audit event, logging any persistence error instead of
// returning it — audit failure must never block the primary auth flow.
func (s *googleServiceImpl) writeAudit(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write google audit event", "event_type", event.EventType, "error", err)
	}
}
