// internal/handler/mfa_handler.go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
	"tigersoft/auth-system/pkg/apierror"
)

// MFAHandler handles TOTP enrollment, confirmation, and disable endpoints.
// All routes require an authenticated JWT (the authed middleware group).
type MFAHandler struct {
	mfaSvc service.MFAService
}

// NewMFAHandler constructs an MFAHandler with its service dependency.
func NewMFAHandler(mfaSvc service.MFAService) *MFAHandler {
	return &MFAHandler{mfaSvc: mfaSvc}
}

type confirmMFARequest struct {
	Secret string `json:"secret" validate:"required,min=16"`
	Code   string `json:"code"   validate:"required,len=6"`
}

type disableMFARequest struct {
	Password string `json:"password" validate:"required"`
}

// Generate handles POST /api/v1/users/me/mfa/generate.
// Creates a new TOTP secret and returns the otpauth URL + base32 secret.
// The secret is NOT persisted until the client calls /mfa/confirm.
func (h *MFAHandler) Generate(c *gin.Context) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}
	claims := claimsVal.(middleware.JWTClaims)

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_USER_ID", "User ID in token is not a valid UUID.", nil, getRequestID(c),
		))
		return
	}

	result, err := h.mfaSvc.GenerateTOTP(c.Request.Context(), userID, claims.UserID, claims.TenantID)
	if err != nil {
		if errors.Is(err, domain.ErrMFAAlreadyEnabled) {
			c.AbortWithStatusJSON(http.StatusConflict, apierror.New(
				"MFA_ALREADY_ENABLED",
				"MFA is already enabled for this account. Disable it first before re-enrolling.",
				nil,
				getRequestID(c),
			))
			return
		}
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"otp_auth_url": result.OTPAuthURL,
		"secret":       result.Secret,
		"message":      "Scan the QR code or enter the secret in your authenticator app, then call /mfa/confirm with a valid code.",
	})
}

// Confirm handles POST /api/v1/users/me/mfa/confirm.
// Validates the first TOTP code to complete enrollment and returns backup codes.
func (h *MFAHandler) Confirm(c *gin.Context) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}
	claims := claimsVal.(middleware.JWTClaims)

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_USER_ID", "User ID in token is not a valid UUID.", nil, getRequestID(c),
		))
		return
	}

	var req confirmMFARequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.mfaSvc.ConfirmTOTP(c.Request.Context(), service.ConfirmTOTPInput{
		UserID: userID,
		Secret: req.Secret,
		Code:   req.Code,
	})
	if err != nil {
		if errors.Is(err, domain.ErrMFAAlreadyEnabled) {
			c.AbortWithStatusJSON(http.StatusConflict, apierror.New(
				"MFA_ALREADY_ENABLED",
				"MFA is already enabled for this account.",
				nil,
				getRequestID(c),
			))
			return
		}
		if errors.Is(err, domain.ErrInvalidTOTPCode) {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apierror.New(
				"INVALID_TOTP_CODE",
				"The TOTP code is invalid or has expired. Check your authenticator app and try again.",
				nil,
				getRequestID(c),
			))
			return
		}
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backup_codes": result.BackupCodes,
		"message":      "MFA has been enabled. Save these backup codes securely — they will not be shown again.",
	})
}

// Disable handles DELETE /api/v1/users/me/mfa.
// Removes TOTP and backup codes after verifying the user's current password.
func (h *MFAHandler) Disable(c *gin.Context) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}
	claims := claimsVal.(middleware.JWTClaims)

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_USER_ID", "User ID in token is not a valid UUID.", nil, getRequestID(c),
		))
		return
	}

	var req disableMFARequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.mfaSvc.DisableMFA(c.Request.Context(), userID, req.Password); err != nil {
		if errors.Is(err, domain.ErrMFANotEnabled) {
			c.AbortWithStatusJSON(http.StatusConflict, apierror.New(
				"MFA_NOT_ENABLED",
				"MFA is not enabled for this account.",
				nil,
				getRequestID(c),
			))
			return
		}
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"INVALID_CREDENTIALS",
				"The password provided is incorrect.",
				nil,
				getRequestID(c),
			))
			return
		}
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
