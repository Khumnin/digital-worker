// internal/handler/auth_handler.go
package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/pkg/apierror"
	"tigersoft/auth-system/internal/service"
)

// AuthHandler handles HTTP requests for core authentication flows.
type AuthHandler struct {
	authSvc service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type registerRequest struct {
	Email     string `json:"email"      validate:"required,email"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
}

type registerResponse struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{
		UserID:  result.UserID,
		Email:   result.Email,
		Status:  result.Status,
		Message: "Registration successful. Please check your email to verify your account.",
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken:      result.AccessToken,
		RefreshToken:     result.RefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        result.ExpiresIn,
		RefreshExpiresIn: result.RefreshExpiresIn,
	})
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.authSvc.Logout(c.Request.Context(), service.LogoutInput{
		RefreshToken: req.RefreshToken,
		IPAddress:    c.ClientIP(),
	}); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully."})
}

// LogoutAll handles POST /api/v1/auth/logout/all
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := mustGetUserID(c)

	revokedCount, err := h.authSvc.LogoutAll(c.Request.Context(), service.LogoutAllInput{
		UserID:    userID,
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "All sessions revoked successfully.",
		"sessions_revoked": revokedCount,
	})
}

// bindAndValidate decodes and validates the JSON request body.
func bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_REQUEST",
			"Request body is invalid or missing required fields.",
			nil,
			getRequestID(c),
		))
		return false
	}

	if errs := validate(req); len(errs) > 0 {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apierror.New(
			"VALIDATION_ERROR",
			"One or more fields failed validation.",
			errs,
			getRequestID(c),
		))
		return false
	}

	return true
}

// respondWithServiceError maps domain/service errors to HTTP responses.
func respondWithServiceError(c *gin.Context, err error) {
	type mapping struct {
		status int
		code   string
		msg    string
	}

	var m mapping

	switch {
	case isError(err, "user not found"), isError(err, "invalid credentials"):
		m = mapping{http.StatusUnauthorized, "INVALID_CREDENTIALS", "The email or password is incorrect."}
	case isError(err, "email not verified"):
		m = mapping{http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Your email address has not been verified. Please check your inbox."}
	case isError(err, "account disabled"):
		m = mapping{http.StatusForbidden, "ACCOUNT_DISABLED", "This account has been disabled."}
	case isError(err, "account is temporarily locked"):
		m = mapping{http.StatusForbidden, "ACCOUNT_LOCKED", "This account is temporarily unavailable."}
	case isError(err, "email already exists"):
		m = mapping{http.StatusConflict, "EMAIL_ALREADY_EXISTS", "An account with this email already exists."}
	case isError(err, "tenant not found"), isError(err, "tenant is suspended"):
		m = mapping{http.StatusForbidden, "INVALID_TENANT", "Tenant not found or is not active."}
	case isError(err, "tenant with this slug already exists"):
		m = mapping{http.StatusConflict, "TENANT_ALREADY_EXISTS", "A tenant with this identifier already exists."}
	case isError(err, "token reuse detected"):
		m = mapping{http.StatusUnauthorized, "SUSPICIOUS_TOKEN_REUSE", "Your session has been revoked due to suspicious activity. Please log in again."}
	case isError(err, "session not found"), isError(err, "session has expired"), isError(err, "invalid refresh token"), isError(err, "session has been revoked"):
		m = mapping{http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Your session has expired. Please log in again."}
	case isError(err, "password does not meet complexity requirements"):
		m = mapping{http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error()}
	case isError(err, "role not found"):
		m = mapping{http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found in this tenant."}
	case isError(err, "role already exists"):
		m = mapping{http.StatusConflict, "ROLE_ALREADY_EXISTS", "A role with this name already exists."}
	case isError(err, "user already has this role"):
		m = mapping{http.StatusConflict, "ROLE_ALREADY_ASSIGNED", "The user already has this role."}
	default:
		c.Error(err)
		m = mapping{http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred. Please try again later."}
	}

	c.AbortWithStatusJSON(m.status, apierror.New(m.code, m.msg, nil, getRequestID(c)))
}

func mustGetUserID(c *gin.Context) string {
	userID, ok := c.Get("user_id")
	if !ok {
		panic("mustGetUserID called without AuthMiddleware")
	}
	return userID.(string)
}

func getRequestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		return id.(string)
	}
	return ""
}

func isError(err error, needle string) bool {
	return err != nil && len(err.Error()) > 0 && strings.Contains(strings.ToLower(err.Error()), strings.ToLower(needle))
}

func validate(v interface{}) []map[string]string {
	return globalValidator.ValidateStruct(v)
}
