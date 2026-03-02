// internal/handler/user_handler.go
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

// UserHandler handles the authenticated user's own profile endpoints.
type UserHandler struct {
	profileSvc service.ProfileService
	mfaSvc     service.MFAService
}

// NewUserHandler constructs a UserHandler with its required service dependencies.
func NewUserHandler(profileSvc service.ProfileService, mfaSvc service.MFAService) *UserHandler {
	return &UserHandler{
		profileSvc: profileSvc,
		mfaSvc:     mfaSvc,
	}
}

// updateMeRequest is the union of all fields a user can send to PUT /users/me.
// The handler dispatches to the appropriate sub-operation based on which fields
// are populated.
type updateMeRequest struct {
	// Display name fields
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`

	// Password change fields
	CurrentPassword *string `json:"current_password"`
	NewPassword     *string `json:"new_password"`

	// Email change field
	NewEmail *string `json:"new_email"`
}

// GetMe handles GET /api/v1/users/me.
// Returns the user's full profile by fetching from the database using the JWT
// subject claim.
func (h *UserHandler) GetMe(c *gin.Context) {
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

	profile, err := h.profileSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, apierror.New(
				"USER_NOT_FOUND", "User profile not found.", nil, getRequestID(c),
			))
			return
		}
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     profile.UserID.String(),
		"email":       profile.Email,
		"first_name":  profile.FirstName,
		"last_name":   profile.LastName,
		"mfa_enabled": profile.MFAEnabled,
		"created_at":  profile.CreatedAt,
		"tenant_id":   claims.TenantID,
		"roles":       claims.Roles,
	})
}

// UpdateMe handles PUT /api/v1/users/me.
// Dispatches to the appropriate ProfileService method based on request fields:
//   - first_name / last_name          -> UpdateProfile
//   - current_password + new_password -> ChangePassword
//   - new_email                       -> RequestEmailChange
func (h *UserHandler) UpdateMe(c *gin.Context) {
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

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_REQUEST", "Request body is invalid or missing required fields.", nil, getRequestID(c),
		))
		return
	}

	// Password change takes precedence when current_password + new_password are present.
	if req.CurrentPassword != nil && req.NewPassword != nil {
		if err := h.profileSvc.ChangePassword(c.Request.Context(), service.ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: *req.CurrentPassword,
			NewPassword:     *req.NewPassword,
			IPAddress:       c.ClientIP(),
		}); err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
					"INVALID_CREDENTIALS",
					"The current password is incorrect.",
					nil,
					getRequestID(c),
				))
				return
			}
			respondWithServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Password changed successfully. All other sessions have been revoked.",
		})
		return
	}

	// Email change when new_email is provided.
	if req.NewEmail != nil {
		if err := h.profileSvc.RequestEmailChange(c.Request.Context(), service.EmailChangeInput{
			UserID:   userID,
			NewEmail: *req.NewEmail,
			TenantID: claims.TenantID,
		}); err != nil {
			if errors.Is(err, domain.ErrEmailAlreadyExists) {
				c.AbortWithStatusJSON(http.StatusConflict, apierror.New(
					"EMAIL_ALREADY_EXISTS",
					"An account with this email already exists.",
					nil,
					getRequestID(c),
				))
				return
			}
			if errors.Is(err, domain.ErrInvalidEmail) {
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apierror.New(
					"INVALID_EMAIL",
					"The provided email address is invalid.",
					nil,
					getRequestID(c),
				))
				return
			}
			respondWithServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "A verification email has been sent to the new address. Complete verification to apply the change.",
		})
		return
	}

	// Display name update.
	if req.FirstName != nil || req.LastName != nil {
		profile, err := h.profileSvc.UpdateProfile(c.Request.Context(), service.UpdateProfileInput{
			UserID:    userID,
			FirstName: req.FirstName,
			LastName:  req.LastName,
		})
		if err != nil {
			respondWithServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":     profile.UserID.String(),
			"email":       profile.Email,
			"first_name":  profile.FirstName,
			"last_name":   profile.LastName,
			"mfa_enabled": profile.MFAEnabled,
			"created_at":  profile.CreatedAt,
		})
		return
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
		"INVALID_REQUEST",
		"No recognised update fields provided. Supply first_name, last_name, current_password+new_password, or new_email.",
		nil,
		getRequestID(c),
	))
}
