// internal/handler/user_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
)

// UserHandler handles the authenticated user's own profile endpoints.
type UserHandler struct {
	authSvc service.AuthService
}

// NewUserHandler constructs a UserHandler with its required service dependency.
func NewUserHandler(svc service.AuthService) *UserHandler {
	return &UserHandler{authSvc: svc}
}

type updateMeRequest struct {
	FirstName string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"omitempty,min=1,max=100"`
}

// GetMe handles GET /api/v1/users/me.
// Returns the identity claims embedded in the caller's JWT.
func (h *UserHandler) GetMe(c *gin.Context) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required.",
			},
		})
		return
	}

	claims := claimsVal.(middleware.JWTClaims)

	c.JSON(http.StatusOK, gin.H{
		"user_id":   claims.UserID,
		"tenant_id": claims.TenantID,
		"roles":     claims.Roles,
	})
}

// UpdateMe handles PUT /api/v1/users/me.
// Full implementation is deferred to Sprint 7 (User Profile story).
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req updateMeRequest
	if !bindAndValidate(c, &req) {
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "Profile update is not yet available.",
		},
	})
}
