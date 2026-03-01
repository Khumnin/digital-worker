// internal/handler/email_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/service"
)

// EmailHandler handles email verification endpoints.
type EmailHandler struct {
	emailSvc service.EmailVerificationService
}

// NewEmailHandler constructs an EmailHandler with its required service dependency.
func NewEmailHandler(svc service.EmailVerificationService) *EmailHandler {
	return &EmailHandler{emailSvc: svc}
}

type verifyEmailRequest struct {
	Token string `json:"token" validate:"required,min=1"`
}

type resendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *EmailHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.emailSvc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully."})
}

// ResendVerification handles POST /api/v1/auth/resend-verification.
// Always returns 200 to prevent email enumeration attacks.
func (h *EmailHandler) ResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// Discard the error intentionally — never reveal whether the email exists.
	_ = h.emailSvc.ResendVerification(c.Request.Context(), req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a new verification email has been sent."})
}
