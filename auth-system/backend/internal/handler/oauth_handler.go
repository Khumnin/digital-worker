// internal/handler/oauth_handler.go
// Sprint 5 placeholder — OAuth 2.0 authorization server endpoints are not yet implemented.
// The OAuthHandler struct has no constructor; main.go holds a nil *OAuthHandler in deps,
// so none of these methods must dereference the receiver before returning.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OAuthHandler handles OAuth 2.0 authorization server endpoints.
// Full implementation using ory/fosite is deferred to Sprint 5.
type OAuthHandler struct{}

// RegisterClient handles POST /api/v1/admin/oauth/clients.
func (h *OAuthHandler) RegisterClient(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth client registration is not yet available.",
		},
	})
}

// Authorize handles GET /api/v1/oauth/authorize.
func (h *OAuthHandler) Authorize(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth authorization endpoint is not yet available.",
		},
	})
}

// Token handles POST /api/v1/oauth/token.
func (h *OAuthHandler) Token(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token endpoint is not yet available.",
		},
	})
}

// Introspect handles POST /api/v1/oauth/introspect.
func (h *OAuthHandler) Introspect(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token introspection is not yet available.",
		},
	})
}

// Revoke handles POST /api/v1/oauth/revoke.
func (h *OAuthHandler) Revoke(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "OAuth token revocation is not yet available.",
		},
	})
}
