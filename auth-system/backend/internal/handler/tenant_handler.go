// internal/handler/tenant_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
)

// TenantHandler handles platform-level tenant lifecycle endpoints.
// All routes under this handler require super-admin privileges.
type TenantHandler struct {
	tenantSvc service.TenantService
}

// NewTenantHandler constructs a TenantHandler with its required service dependency.
func NewTenantHandler(svc service.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: svc}
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
		"tenant_id":   tenant.ID.String(),
		"name":        tenant.Name,
		"slug":        tenant.Slug,
		"schema_name": tenant.SchemaName,
		"status":      string(tenant.Status),
		"created_at":  tenant.CreatedAt,
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
		"tenant_id":   tenant.ID.String(),
		"name":        tenant.Name,
		"slug":        tenant.Slug,
		"schema_name": tenant.SchemaName,
		"status":      string(tenant.Status),
		"created_at":  tenant.CreatedAt,
	})
}

// ListTenants handles GET /api/v1/admin/tenants.
// Supports pagination via limit and offset query parameters.
func (h *TenantHandler) ListTenants(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	tenants, total, err := h.tenantSvc.ListTenants(c.Request.Context(), limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(tenants))
	for i, t := range tenants {
		items[i] = gin.H{
			"tenant_id":  t.ID.String(),
			"name":       t.Name,
			"slug":       t.Slug,
			"status":     string(t.Status),
			"created_at": t.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// SuspendTenant handles PUT /api/v1/admin/tenants/:id/suspend.
// Marks the tenant as suspended so all logins for that tenant are rejected.
func (h *TenantHandler) SuspendTenant(c *gin.Context) {
	id := c.Param("id")

	if err := h.tenantSvc.SuspendTenant(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant has been suspended."})
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Request body is invalid.",
			},
		})
		return
	}

	// middleware.RequireAuth stores a middleware.JWTClaims value under the
	// "jwt_claims" key. Extract the TenantID from it directly.
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required."},
		})
		return
	}

	claims, ok := claimsVal.(middleware.JWTClaims)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "Unexpected claims type in context."},
		})
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
