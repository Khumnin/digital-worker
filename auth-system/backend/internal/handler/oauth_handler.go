// internal/handler/oauth_handler.go
// Sprint 4: Token Introspection (M17, RFC 7662).
// Sprint 5: Authorization Code + PKCE flow (US-11a, US-11b, US-11c).
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
	"tigersoft/auth-system/pkg/jwtutil"
)

// OAuthHandler handles OAuth 2.0 authorization server endpoints.
type OAuthHandler struct {
	verifier jwtutil.Verifier
	oauthSvc service.OAuthService
}

// NewOAuthHandler constructs an OAuthHandler.
func NewOAuthHandler(verifier jwtutil.Verifier, oauthSvc service.OAuthService) *OAuthHandler {
	return &OAuthHandler{verifier: verifier, oauthSvc: oauthSvc}
}

// ── US-11a: Client Registration ───────────────────────────────────────────────

type registerClientRequest struct {
	Name         string   `json:"name"          binding:"required"`
	RedirectURIs []string `json:"redirect_uris" binding:"required,min=1"`
	Scopes       []string `json:"scopes"`
}

// RegisterClient handles POST /api/v1/admin/oauth/clients.
func (h *OAuthHandler) RegisterClient(c *gin.Context) {
	var req registerClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	adminID := mustOAuthUserID(c)
	result, err := h.oauthSvc.RegisterClient(c.Request.Context(), service.RegisterClientInput{
		Name:         req.Name,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		CreatedBy:    adminID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":     result.ClientID,
		"client_secret": result.ClientSecret,
		"name":          result.Name,
		"warning":       "Store client_secret securely — it will not be shown again.",
	})
}

// ── US-11b: Authorization Endpoint ───────────────────────────────────────────

// Authorize handles GET /api/v1/oauth/authorize.
// Requires a valid user session (authMW already enforced by router).
func (h *OAuthHandler) Authorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	scope := c.Query("scope")
	state := c.Query("state")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.Query("code_challenge_method")

	if clientID == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "INVALID_REQUEST", "message": "client_id and redirect_uri are required"},
		})
		return
	}

	userID := mustOAuthUserID(c)
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	result, err := h.oauthSvc.Authorize(c.Request.Context(), service.AuthorizeInput{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        responseType,
		Scope:               scope,
		State:               state,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		UserID:              userID,
		TenantID:            tenantIDStr,
	})
	if err != nil {
		status := http.StatusBadRequest
		code := "INVALID_REQUEST"
		switch {
		case errors.Is(err, domain.ErrOAuthClientNotFound):
			// Per spec: do NOT redirect on unknown client_id — return error directly
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_CLIENT", "message": "unknown client_id"},
			})
			return
		case errors.Is(err, domain.ErrInvalidRedirectURI):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "INVALID_REDIRECT_URI", "message": err.Error()},
			})
			return
		case errors.Is(err, domain.ErrUnsupportedResponseType):
			code = "UNSUPPORTED_RESPONSE_TYPE"
		case errors.Is(err, domain.ErrStateMissing):
			code = "INVALID_REQUEST"
		case errors.Is(err, domain.ErrInvalidCodeChallenge):
			code = "INVALID_REQUEST"
		case errors.Is(err, domain.ErrInvalidScope):
			code = "INVALID_SCOPE"
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": gin.H{"code": code, "message": err.Error()}})
		return
	}

	// API-only per ADR-003 — return code in JSON body; client handles redirect
	c.JSON(http.StatusOK, gin.H{
		"code":  result.Code,
		"state": result.State,
	})
}

// ── US-11c: Token Exchange ────────────────────────────────────────────────────

type tokenRequest struct {
	GrantType    string `json:"grant_type"    form:"grant_type"    binding:"required"`
	Code         string `json:"code"          form:"code"`
	CodeVerifier string `json:"code_verifier" form:"code_verifier"`
	ClientID     string `json:"client_id"     form:"client_id"     binding:"required"`
	RedirectURI  string `json:"redirect_uri"  form:"redirect_uri"  binding:"required"`
}

// Token handles POST /api/v1/oauth/token (authorization_code grant + PKCE).
func (h *OAuthHandler) Token(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid_request", "error_description": err.Error(),
		})
		return
	}
	if req.GrantType != "authorization_code" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported_grant_type",
			"error_description": "only authorization_code grant type is supported",
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	result, err := h.oauthSvc.ExchangeToken(c.Request.Context(), service.ExchangeTokenInput{
		Code:         req.Code,
		CodeVerifier: req.CodeVerifier,
		ClientID:     req.ClientID,
		RedirectURI:  req.RedirectURI,
		TenantID:     tenantIDStr,
	})
	if err != nil {
		errCode := "invalid_grant"
		switch {
		case errors.Is(err, domain.ErrPKCEVerificationFailed):
			errCode = "invalid_grant"
		case errors.Is(err, domain.ErrAuthCodeAlreadyUsed):
			errCode = "invalid_grant"
		case errors.Is(err, domain.ErrAuthCodeExpired):
			errCode = "invalid_grant"
		case errors.Is(err, domain.ErrInvalidRedirectURI):
			errCode = "invalid_grant"
		case errors.Is(err, domain.ErrAuthCodeNotFound):
			errCode = "invalid_grant"
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errCode, "error_description": err.Error(),
		})
		return
	}

	// RFC 6749 §5.1 token response
	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    result.TokenType,
		"expires_in":    result.ExpiresIn,
		"scope":         result.Scope,
	})
}

// introspectRequest accepts either JSON or form-encoded body per RFC 7662 §2.1.
type introspectRequest struct {
	Token string `json:"token" form:"token"`
}

// Introspect handles POST /api/v1/oauth/introspect (RFC 7662).
// Returns {"active": true, ...claims} for valid tokens, {"active": false} for all others.
// Does NOT require authentication so resource servers can call it without a bearer token.
func (h *OAuthHandler) Introspect(c *gin.Context) {
	var req introspectRequest
	// Accept both application/json and application/x-www-form-urlencoded (RFC 7662 §2.1)
	if err := c.ShouldBind(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	claims, err := h.verifier.Verify(req.Token)
	if err != nil {
		// Any error (expired, bad signature, wrong audience) → inactive
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"token_type": "Bearer",
		"sub":        claims.Subject,
		"jti":        claims.ID,
		"iss":        c.Request.Host, // issuer from verified token context
		"iat":        claims.IssuedAt.Unix(),
		"exp":        claims.ExpiresAt.Unix(),
		"tenant_id":  claims.TenantID,
		"roles":      claims.Roles,
		"scope":      claims.Scope,
		"client_id":  claims.ClientID,
	})
}

// Revoke handles POST /api/v1/oauth/revoke.
// Sprint 5 — deferred to future sprint (full session revocation already supported via /auth/logout).
func (h *OAuthHandler) Revoke(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token revocation is not yet available.",
		},
	})
}

// mustOAuthUserID extracts the authenticated user's UUID from Gin context.
// Panics if called outside of the auth middleware chain.
func mustOAuthUserID(c *gin.Context) uuid.UUID {
	raw, ok := c.Get("user_id")
	if !ok {
		panic("mustOAuthUserID called without auth middleware")
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		panic("mustOAuthUserID: invalid UUID in context: " + err.Error())
	}
	return id
}
