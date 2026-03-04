// internal/service/oauth_service.go
// Sprint 5 — OAuth 2.0 Authorization Code + PKCE flow (US-11a, US-11b, US-11c).
// No external OAuth library is used; PKCE S256 and code exchange are implemented
// directly using the existing jwt and crypto utilities.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/pkg/crypto"
	"tigersoft/auth-system/pkg/jwtutil"
)

const (
	authCodeTTL     = 10 * time.Minute
	oauthGrantType  = "authorization_code"
	pkceMethodS256  = "S256"
)

// OAuthService handles the OAuth 2.0 Authorization Code + PKCE flow,
// and the client_credentials grant for M2M access (Sprint 6 US-12).
type OAuthService interface {
	// RegisterClient creates a new OAuth 2.0 client for the calling admin.
	RegisterClient(ctx context.Context, input RegisterClientInput) (*RegisterClientResult, error)

	// Authorize validates an authorization request and issues a one-time authorization code.
	Authorize(ctx context.Context, input AuthorizeInput) (*AuthorizeResult, error)

	// ExchangeToken validates a code + PKCE verifier and returns JWT access + refresh tokens.
	ExchangeToken(ctx context.Context, input ExchangeTokenInput) (*ExchangeTokenResult, error)

	// M2MToken issues an access token for the client_credentials grant (no user, no refresh token).
	M2MToken(ctx context.Context, input M2MTokenInput) (*M2MTokenResult, error)
}

type oauthServiceImpl struct {
	clientRepo  domain.OAuthClientRepository
	codeRepo    domain.AuthorizationCodeRepository
	sessionRepo domain.SessionRepository
	auditRepo   domain.AuditRepository
	jwtSvc      jwtutil.Signer
	accessTTL   time.Duration
	sessionTTL  time.Duration
}

func NewOAuthService(
	clientRepo domain.OAuthClientRepository,
	codeRepo domain.AuthorizationCodeRepository,
	sessionRepo domain.SessionRepository,
	auditRepo domain.AuditRepository,
	jwtSvc jwtutil.Signer,
	accessTTL time.Duration,
	sessionTTL time.Duration,
) OAuthService {
	return &oauthServiceImpl{
		clientRepo:  clientRepo,
		codeRepo:    codeRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		jwtSvc:      jwtSvc,
		accessTTL:   accessTTL,
		sessionTTL:  sessionTTL,
	}
}

// ── US-11a: Client Registration ───────────────────────────────────────────────

type RegisterClientInput struct {
	Name         string
	RedirectURIs []string
	Scopes       []string
	CreatedBy    uuid.UUID
}

type RegisterClientResult struct {
	ClientID     string
	ClientSecret string // plaintext — only returned once; caller must store it
	Name         string
}

func (s *oauthServiceImpl) RegisterClient(ctx context.Context, in RegisterClientInput) (*RegisterClientResult, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("client name is required")
	}
	if len(in.RedirectURIs) == 0 {
		return nil, fmt.Errorf("at least one redirect_uri is required")
	}
	for _, u := range in.RedirectURIs {
		if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://localhost") {
			return nil, fmt.Errorf("redirect_uri must use HTTPS (or http://localhost for development): %s", u)
		}
	}
	if len(in.Scopes) == 0 {
		in.Scopes = []string{"openid"}
	}

	rawSecret, secretHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate client secret: %w", err)
	}
	rawClientID, _, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate client id: %w", err)
	}
	clientID := rawClientID[:32] // 32-char hex prefix is enough for a client_id

	client := &domain.OAuthClient{
		ID:             uuid.New(),
		ClientID:       clientID,
		ClientSecret:   secretHash,
		Name:           in.Name,
		RedirectURIs:   in.RedirectURIs,
		Scopes:         in.Scopes,
		GrantTypes:     []string{oauthGrantType},
		IsConfidential: true,
		CreatedBy:      &in.CreatedBy,
	}

	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, fmt.Errorf("store oauth client: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType: domain.EventOAuthClientCreated,
		ActorID:   &in.CreatedBy,
		Metadata:  map[string]interface{}{"client_id": clientID, "name": in.Name},
	})

	return &RegisterClientResult{
		ClientID:     clientID,
		ClientSecret: rawSecret,
		Name:         in.Name,
	}, nil
}

// ── US-11b: Authorization Endpoint ───────────────────────────────────────────

type AuthorizeInput struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              uuid.UUID
	TenantID            string
}

type AuthorizeResult struct {
	Code  string
	State string
}

func (s *oauthServiceImpl) Authorize(ctx context.Context, in AuthorizeInput) (*AuthorizeResult, error) {
	if in.ResponseType != "code" {
		return nil, domain.ErrUnsupportedResponseType
	}
	if in.State == "" {
		return nil, domain.ErrStateMissing
	}
	if in.CodeChallenge == "" || strings.ToUpper(in.CodeChallengeMethod) != pkceMethodS256 {
		return nil, domain.ErrInvalidCodeChallenge
	}

	client, err := s.clientRepo.FindByClientID(ctx, in.ClientID)
	if err != nil {
		return nil, domain.ErrOAuthClientNotFound
	}
	if !client.HasRedirectURI(in.RedirectURI) {
		return nil, domain.ErrInvalidRedirectURI
	}

	requestedScopes := parseScopes(in.Scope)
	for _, rs := range requestedScopes {
		if !client.HasScope(rs) {
			return nil, domain.ErrInvalidScope
		}
	}

	rawCode, codeHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate authorization code: %w", err)
	}

	code := &domain.AuthorizationCode{
		ID:                  uuid.New(),
		CodeHash:            codeHash,
		ClientID:            in.ClientID,
		UserID:              in.UserID,
		RedirectURI:         in.RedirectURI,
		Scopes:              requestedScopes,
		CodeChallenge:       in.CodeChallenge,
		CodeChallengeMethod: pkceMethodS256,
		ExpiresAt:           time.Now().Add(authCodeTTL),
	}
	if err := s.codeRepo.Create(ctx, code); err != nil {
		return nil, fmt.Errorf("store authorization code: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType: domain.EventOAuthCodeIssued,
		ActorID:   &in.UserID,
		Metadata:  map[string]interface{}{"client_id": in.ClientID},
	})

	return &AuthorizeResult{Code: rawCode, State: in.State}, nil
}

// ── US-11c: Token Exchange ────────────────────────────────────────────────────

type ExchangeTokenInput struct {
	Code         string
	CodeVerifier string
	ClientID     string
	RedirectURI  string
	TenantID     string
	UserRoles    []string
}

type ExchangeTokenResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Scope        string
}

func (s *oauthServiceImpl) ExchangeToken(ctx context.Context, in ExchangeTokenInput) (*ExchangeTokenResult, error) {
	if in.Code == "" || in.CodeVerifier == "" {
		return nil, fmt.Errorf("code and code_verifier are required")
	}

	codeHash := crypto.HashTokenString(in.Code)
	code, err := s.codeRepo.FindByCodeHash(ctx, codeHash)
	if err != nil {
		return nil, domain.ErrAuthCodeNotFound
	}
	if code.Used {
		// Code reuse detected — mark used (already is) and fail
		return nil, domain.ErrAuthCodeAlreadyUsed
	}
	if code.IsExpired() {
		return nil, domain.ErrAuthCodeExpired
	}
	if code.ClientID != in.ClientID {
		return nil, domain.ErrAuthCodeNotFound
	}
	if code.RedirectURI != in.RedirectURI {
		return nil, domain.ErrInvalidRedirectURI
	}

	// PKCE S256 verification: SHA-256(code_verifier) base64url == code_challenge
	if !verifyPKCES256(in.CodeVerifier, code.CodeChallenge) {
		return nil, domain.ErrPKCEVerificationFailed
	}

	// Mark code used before issuing tokens (prevents replay)
	if err := s.codeRepo.MarkUsed(ctx, codeHash); err != nil {
		return nil, fmt.Errorf("mark code as used: %w", err)
	}

	roles := in.UserRoles
	if len(roles) == 0 {
		roles = []string{"user"}
	}

	accessToken, err := s.jwtSvc.Sign(jwtutil.Claims{
		Subject:  code.UserID.String(),
		TenantID: in.TenantID,
		Roles:    roles,
		Scope:    strings.Join(code.Scopes, " "),
		ClientID: in.ClientID,
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
		UserID:           code.UserID,
		RefreshTokenHash: refreshHash,
		FamilyID:         uuid.New(),
		IssuedAt:         time.Now(),
		ExpiresAt:        time.Now().Add(s.sessionTTL),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType: domain.EventOAuthTokenIssued,
		ActorID:   &code.UserID,
		Metadata:  map[string]interface{}{"client_id": in.ClientID},
	})

	return &ExchangeTokenResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
		Scope:        strings.Join(code.Scopes, " "),
	}, nil
}

// ── US-12: M2M Token (client_credentials grant) ───────────────────────────────

// M2MTokenInput carries the credentials and scope for a client_credentials grant.
type M2MTokenInput struct {
	ClientID     string
	ClientSecret string
	Scope        string
	TenantID     string
}

// M2MTokenResult holds the issued access token for a machine-to-machine client.
// Per RFC 6749 §4.4.3 there is no refresh token for this grant type.
type M2MTokenResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
	Scope       string
}

func (s *oauthServiceImpl) M2MToken(ctx context.Context, in M2MTokenInput) (*M2MTokenResult, error) {
	client, err := s.clientRepo.FindByClientID(ctx, in.ClientID)
	if err != nil {
		// Obscure the exact reason to prevent client enumeration.
		return nil, domain.ErrInvalidClientCredentials
	}

	// Verify the client supports client_credentials.
	hasGrant := false
	for _, g := range client.GrantTypes {
		if g == "client_credentials" {
			hasGrant = true
			break
		}
	}
	if !hasGrant {
		return nil, domain.ErrClientCredentialsGrantNotAllowed
	}

	// Verify secret: compare stored hash against the hash of the supplied secret.
	if crypto.HashTokenString(in.ClientSecret) != client.ClientSecret {
		return nil, domain.ErrInvalidClientCredentials
	}

	// Validate requested scopes against the client's allowed scopes.
	requestedScopes := parseScopes(in.Scope)
	for _, rs := range requestedScopes {
		if !client.HasScope(rs) {
			return nil, domain.ErrInvalidScope
		}
	}

	// Issue a JWT whose subject is the client_id — no user identity involved.
	accessToken, err := s.jwtSvc.Sign(jwtutil.Claims{
		Subject:  in.ClientID,
		TenantID: in.TenantID,
		Roles:    []string{},
		Scope:    strings.Join(requestedScopes, " "),
		ClientID: in.ClientID,
		TTL:      s.accessTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("sign M2M access token: %w", err)
	}

	s.writeAudit(ctx, domain.AuditEvent{
		EventType: domain.EventOAuthTokenIssued,
		Metadata: map[string]interface{}{
			"client_id":  in.ClientID,
			"grant_type": "client_credentials",
		},
	})

	return &M2MTokenResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTTL.Seconds()),
		Scope:       strings.Join(requestedScopes, " "),
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// verifyPKCES256 checks that SHA-256(verifier) == base64url(challenge) per RFC 7636.
func verifyPKCES256(verifier, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return computed == challenge
}

func parseScopes(scope string) []string {
	if scope == "" {
		return []string{"openid"}
	}
	parts := strings.Fields(scope)
	seen := make(map[string]bool, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func (s *oauthServiceImpl) writeAudit(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write oauth audit event", "event_type", event.EventType, "error", err)
	}
}
