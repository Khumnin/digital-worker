// internal/handler/password_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/service"
)

// PasswordHandler handles forgot-password and reset-password endpoints.
type PasswordHandler struct {
	passwordSvc service.PasswordService
}

// NewPasswordHandler constructs a PasswordHandler with its required service dependency.
func NewPasswordHandler(svc service.PasswordService) *PasswordHandler {
	return &PasswordHandler{passwordSvc: svc}
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"        validate:"required,min=1"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
// Always returns 200 to prevent email enumeration attacks.
func (h *PasswordHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// Discard the error intentionally — never reveal whether the email exists.
	_ = h.passwordSvc.ForgotPassword(c.Request.Context(), req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

// ResetPassword handles POST /api/v1/auth/reset-password.
func (h *PasswordHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.passwordSvc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully. Please log in with your new password."})
}
