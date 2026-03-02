// internal/handler/google_handler.go
// Sprint 6 placeholder — Google OAuth social login is not yet implemented.
// The GoogleHandler struct has no constructor; main.go holds a nil *GoogleHandler in deps,
// so none of these methods must dereference the receiver before returning.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GoogleHandler handles the Google OAuth 2.0 social login flow.
// Full implementation is deferred to Sprint 6.
type GoogleHandler struct{}

// Initiate handles POST /api/v1/auth/oauth/google.
// In the full implementation this will redirect the user to Google's consent screen.
func (h *GoogleHandler) Initiate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "Google OAuth login is not yet available.",
		},
	})
}

// Callback handles GET /api/v1/auth/oauth/google/callback.
// In the full implementation this will exchange the authorization code for tokens
// and complete the user login or account-linking flow.
func (h *GoogleHandler) Callback(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "Google OAuth callback is not yet available.",
		},
	})
}
