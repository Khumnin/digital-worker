// internal/handler/oauth_handler.go
// Sprint 4: Token Introspection (M17, RFC 7662) implemented.
// Remaining OAuth 2.0 authorization server endpoints (Authorize, Token, Revoke,
// RegisterClient) are deferred to Sprint 5 (ory/fosite integration).
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/pkg/jwtutil"
)

// OAuthHandler handles OAuth 2.0 authorization server endpoints.
type OAuthHandler struct {
	verifier jwtutil.Verifier
}

// NewOAuthHandler constructs an OAuthHandler with a JWT verifier for introspection.
func NewOAuthHandler(verifier jwtutil.Verifier) *OAuthHandler {
	return &OAuthHandler{verifier: verifier}
}

// RegisterClient handles POST /api/v1/admin/oauth/clients.
// Sprint 5 — deferred to ory/fosite integration.
func (h *OAuthHandler) RegisterClient(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth client registration is not yet available.",
		},
	})
}

// Authorize handles GET /api/v1/oauth/authorize.
// Sprint 5 — deferred to ory/fosite integration.
func (h *OAuthHandler) Authorize(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth authorization endpoint is not yet available.",
		},
	})
}

// Token handles POST /api/v1/oauth/token.
// Sprint 5 — deferred to ory/fosite integration.
func (h *OAuthHandler) Token(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token endpoint is not yet available.",
		},
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
// Sprint 5 — deferred to ory/fosite integration.
func (h *OAuthHandler) Revoke(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token revocation is not yet available.",
		},
	})
}
