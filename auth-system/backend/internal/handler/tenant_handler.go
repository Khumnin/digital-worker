// internal/handler/tenant_handler.go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	pginfra "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
	"tigersoft/auth-system/pkg/apierror"
)

// TenantHandler handles platform-level tenant lifecycle endpoints.
// All routes under this handler require super-admin privileges.
type TenantHandler struct {
	tenantSvc service.TenantService
	adminSvc  service.AdminService
}

// NewTenantHandler constructs a TenantHandler with its required service dependencies.
func NewTenantHandler(svc service.TenantService, adminSvc service.AdminService) *TenantHandler {
	return &TenantHandler{tenantSvc: svc, adminSvc: adminSvc}
}

type provisionTenantRequest struct {
	Name       string `json:"name"        validate:"required,min=2,max=100"`
	Slug       string `json:"slug"        validate:"required,min=3,max=50"`
	AdminEmail string `json:"admin_email" validate:"required,email"`
}

// ProvisionTenant handles POST /api/v1/admin/tenants.
// Creates the tenant record, provisions its PostgreSQL schema, and seeds the
// admin user account in a single atomic operation.
func (h *TenantHandler) ProvisionTenant(c *gin.Context) {
	var req provisionTenantRequest
	if !bindAndValidate(c, &req) {
		return
	}

	tenant, err := h.tenantSvc.ProvisionTenant(c.Request.Context(), service.ProvisionTenantInput{
		Name:       req.Name,
		Slug:       req.Slug,
		AdminEmail: req.AdminEmail,
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":              tenant.ID.String(),
		"name":            tenant.Name,
		"slug":            tenant.Slug,
		"status":          string(tenant.Status),
		"enabled_modules": []string{},
		"created_at":      tenant.CreatedAt,
	})
}

// GetTenant handles GET /api/v1/admin/tenants/:id.
func (h *TenantHandler) GetTenant(c *gin.Context) {
	id := c.Param("id")

	tenant, err := h.tenantSvc.GetTenant(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              tenant.ID.String(),
		"name":            tenant.Name,
		"slug":            tenant.Slug,
		"status":          string(tenant.Status),
		"enabled_modules": []string{},
		"created_at":      tenant.CreatedAt,
	})
}

// ListTenants handles GET /api/v1/admin/tenants.
// Supports pagination via page and page_size query parameters.
func (h *TenantHandler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	tenants, total, err := h.tenantSvc.ListTenants(c.Request.Context(), pageSize, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(tenants))
	for i, t := range tenants {
		items[i] = gin.H{
			"id":              t.ID.String(),
			"name":            t.Name,
			"slug":            t.Slug,
			"status":          string(t.Status),
			"enabled_modules": []string{},
			"created_at":      t.CreatedAt,
		}
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// SuspendTenant handles POST /api/v1/admin/tenants/:id/suspend.
// Marks the tenant as suspended so all logins for that tenant are rejected.
func (h *TenantHandler) SuspendTenant(c *gin.Context) {
	id := c.Param("id")

	if err := h.tenantSvc.SuspendTenant(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant has been suspended."})
}

// ActivateTenant handles POST /api/v1/admin/tenants/:id/activate.
// Moves a suspended or pending tenant back to active status.
func (h *TenantHandler) ActivateTenant(c *gin.Context) {
	id := c.Param("id")

	if err := h.tenantSvc.ActivateTenant(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	tenant, err := h.tenantSvc.GetTenant(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	enabledModules := tenant.Config.EnabledModules
	if enabledModules == nil {
		enabledModules = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              tenant.ID.String(),
		"name":            tenant.Name,
		"slug":            tenant.Slug,
		"status":          string(tenant.Status),
		"enabled_modules": enabledModules,
		"created_at":      tenant.CreatedAt,
		"updated_at":      tenant.UpdatedAt,
	})
}

// GenerateCredentials handles POST /api/v1/admin/tenants/:id/credentials.
// Issues a new client_id and client_secret for M2M authentication.
// The secret is returned once and cannot be retrieved again.
func (h *TenantHandler) GenerateCredentials(c *gin.Context) {
	id := c.Param("id")

	clientID, secret, err := h.tenantSvc.GenerateAPICredentials(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":     clientID,
		"client_secret": secret,
		"warning":       "This is the only time the client_secret will be shown. Store it securely.",
	})
}

// GetTenantSettings handles GET /api/v1/admin/tenant.
// Returns the calling admin's own tenant settings, resolved from the JWT tenant slug.
func (h *TenantHandler) GetTenantSettings(c *gin.Context) {
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}

	claims, ok := claimsVal.(middleware.JWTClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New(
			"INTERNAL_ERROR", "Unexpected claims type in context.", nil, getRequestID(c),
		))
		return
	}

	tenant, err := h.tenantSvc.GetTenantBySlug(c.Request.Context(), claims.TenantID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	allowedDomains := tenant.Config.AllowedCORSOrigins
	if allowedDomains == nil {
		allowedDomains = []string{}
	}
	enabledModules := tenant.Config.EnabledModules
	if enabledModules == nil {
		enabledModules = []string{}
	}

	sessionHours := tenant.Config.SessionTTLSeconds / 3600
	if sessionHours == 0 {
		sessionHours = 24
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                     tenant.ID.String(),
		"name":                   tenant.Name,
		"slug":                   tenant.Slug,
		"status":                 string(tenant.Status),
		"enabled_modules":        enabledModules,
		"mfa_required":           tenant.Config.MFARequired,
		"session_duration_hours": sessionHours,
		"allowed_domains":        allowedDomains,
		"created_at":             tenant.CreatedAt,
		"updated_at":             tenant.UpdatedAt,
	})
}

type updateTenantSettingsRequest struct {
	MFARequired           *bool     `json:"mfa_required"`
	SessionDurationHours  *int      `json:"session_duration_hours"`
	AllowedDomains        []string  `json:"allowed_domains"`
}

// UpdateTenantSettings handles PUT /api/v1/admin/tenant.
// Partial update: only provided fields are changed.
func (h *TenantHandler) UpdateTenantSettings(c *gin.Context) {
	var req updateTenantSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_REQUEST", "Request body is invalid.", nil, getRequestID(c),
		))
		return
	}

	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}

	claims, ok := claimsVal.(middleware.JWTClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New(
			"INTERNAL_ERROR", "Unexpected claims type in context.", nil, getRequestID(c),
		))
		return
	}

	tenant, err := h.tenantSvc.UpdateTenantSettings(c.Request.Context(), claims.TenantID, service.UpdateTenantSettingsInput{
		MFARequired:          req.MFARequired,
		SessionDurationHours: req.SessionDurationHours,
		AllowedDomains:       req.AllowedDomains,
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	allowedDomains := tenant.Config.AllowedCORSOrigins
	if allowedDomains == nil {
		allowedDomains = []string{}
	}
	enabledModules := tenant.Config.EnabledModules
	if enabledModules == nil {
		enabledModules = []string{}
	}

	sessionHours := tenant.Config.SessionTTLSeconds / 3600
	if sessionHours == 0 {
		sessionHours = 24
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                     tenant.ID.String(),
		"name":                   tenant.Name,
		"slug":                   tenant.Slug,
		"status":                 string(tenant.Status),
		"enabled_modules":        enabledModules,
		"mfa_required":           tenant.Config.MFARequired,
		"session_duration_hours": sessionHours,
		"allowed_domains":        allowedDomains,
		"created_at":             tenant.CreatedAt,
		"updated_at":             tenant.UpdatedAt,
	})
}

type updateMFAConfigRequest struct {
	MFARequired bool `json:"mfa_required"`
}

// UpdateMFAConfig handles PUT /api/v1/admin/tenant/mfa.
// Toggles MFA enforcement for the calling admin's tenant.
// The tenant ID is resolved from the JWT claims injected by RequireAuth.
// Requires the admin role — enforced by the router middleware group.
func (h *TenantHandler) UpdateMFAConfig(c *gin.Context) {
	var req updateMFAConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_REQUEST", "Request body is invalid.", nil, getRequestID(c),
		))
		return
	}

	// middleware.RequireAuth stores a middleware.JWTClaims value under the
	// "jwt_claims" key. Extract the TenantID from it directly.
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
			"UNAUTHORIZED", "Authentication required.", nil, getRequestID(c),
		))
		return
	}

	claims, ok := claimsVal.(middleware.JWTClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New(
			"INTERNAL_ERROR", "Unexpected claims type in context.", nil, getRequestID(c),
		))
		return
	}

	if err := h.tenantSvc.UpdateMFARequirement(c.Request.Context(), claims.TenantID, req.MFARequired); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mfa_required": req.MFARequired,
		"message":      "MFA enforcement setting updated.",
	})
}

// RotateCredentials handles POST /api/v1/admin/tenants/:id/credentials/rotate.
// Revokes all existing credentials and issues a new client_id + client_secret.
func (h *TenantHandler) RotateCredentials(c *gin.Context) {
	id := c.Param("id")

	clientID, secret, err := h.tenantSvc.RotateAPICredentials(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":     clientID,
		"client_secret": secret,
		"warning":       "Previous credentials have been revoked. Store the new secret securely.",
	})
}

type inviteAdminRequest struct {
	Email       string `json:"email"        validate:"required,email"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=200"`
}

// InviteAdminToTenant handles POST /api/v1/admin/tenants/:id/invite-admin.
// Creates a new admin user in the target tenant's schema and sends an invitation email.
// The operation runs in the target tenant's PostgreSQL schema, not the caller's.
func (h *TenantHandler) InviteAdminToTenant(c *gin.Context) {
	tenantID := c.Param("id")

	var req inviteAdminRequest
	if !bindAndValidate(c, &req) {
		return
	}

	tenant, err := h.tenantSvc.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	// Operate in the target tenant's schema and identity, not the platform schema.
	ctx := context.WithValue(c.Request.Context(), pginfra.CtxKeySchemaName, tenant.SchemaName)
	ctx = context.WithValue(ctx, pginfra.CtxKeyTenantID, tenant.Slug)

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	user, err := h.adminSvc.InviteUser(ctx, req.Email, req.DisplayName, "", "admin", claims.UserID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           user.ID.String(),
		"email":        user.Email,
		"display_name": user.FullName(),
		"status":       normalizeUserStatus(string(user.Status)),
		"system_roles": []string{"admin"},
		"tenant_id":    tenantID,
		"created_at":   user.CreatedAt,
	})
}

// ListTenantUsers handles GET /api/v1/admin/tenants/:id/users.
// Returns the list of users in the specified tenant's schema (cross-tenant read).
func (h *TenantHandler) ListTenantUsers(c *gin.Context) {
	tenantID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tenant, err := h.tenantSvc.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	// Operate in the target tenant's schema and identity.
	ctx := context.WithValue(c.Request.Context(), pginfra.CtxKeySchemaName, tenant.SchemaName)
	ctx = context.WithValue(ctx, pginfra.CtxKeyTenantID, tenant.Slug)

	users, total, err := h.adminSvc.ListUsers(ctx, pageSize, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(users))
	for i, u := range users {
		displayName := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if displayName == "" {
			displayName = u.Email
		}
		items[i] = gin.H{
			"id":           u.ID.String(),
			"email":        u.Email,
			"display_name": displayName,
			"status":       normalizeUserStatus(string(u.Status)),
			"system_roles": []string{},
			"module_roles": map[string][]string{},
			"tenant_id":    tenantID,
			"created_at":   u.CreatedAt,
		}
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// ResendAdminInvite handles POST /api/v1/admin/tenants/:id/users/:userId/resend-invite.
// Re-sends the invitation email for a pending user in the specified tenant's schema.
func (h *TenantHandler) ResendAdminInvite(c *gin.Context) {
	tenantID := c.Param("id")
	userID := c.Param("userId")

	tenant, err := h.tenantSvc.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	ctx := context.WithValue(c.Request.Context(), pginfra.CtxKeySchemaName, tenant.SchemaName)
	ctx = context.WithValue(ctx, pginfra.CtxKeyTenantID, tenant.Slug)

	if err := h.adminSvc.ResendInvite(ctx, userID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation re-sent successfully."})
}
