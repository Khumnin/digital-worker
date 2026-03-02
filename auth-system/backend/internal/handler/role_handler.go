// internal/handler/role_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
)

// RoleHandler handles RBAC role management and user-role assignment endpoints.
type RoleHandler struct {
	rbacSvc service.RBACService
}

// NewRoleHandler constructs a RoleHandler with its required service dependency.
func NewRoleHandler(svc service.RBACService) *RoleHandler {
	return &RoleHandler{rbacSvc: svc}
}

type createRoleRequest struct {
	Name        string `json:"name"        validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
}

type assignRoleRequest struct {
	RoleID string `json:"role_id" validate:"required,uuid"`
}

// CreateRole handles POST /api/v1/admin/roles.
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if !bindAndValidate(c, &req) {
		return
	}

	role, err := h.rbacSvc.CreateRole(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"role_id":     role.ID.String(),
		"name":        role.Name,
		"description": role.Description,
		"is_system":   role.IsSystem,
		"created_at":  role.CreatedAt,
	})
}

// ListRoles handles GET /api/v1/admin/roles.
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.rbacSvc.ListRoles(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(roles))
	for i, r := range roles {
		items[i] = gin.H{
			"role_id":     r.ID.String(),
			"name":        r.Name,
			"description": r.Description,
			"is_system":   r.IsSystem,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

// AssignRole handles POST /api/v1/admin/users/:id/roles.
// Records the assigning admin's user ID from their JWT claims for the audit trail.
func (h *RoleHandler) AssignRole(c *gin.Context) {
	userID := c.Param("id")

	var req assignRoleRequest
	if !bindAndValidate(c, &req) {
		return
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	if err := h.rbacSvc.AssignRole(c.Request.Context(), userID, req.RoleID, claims.UserID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully."})
}

// UnassignRole handles DELETE /api/v1/admin/users/:id/roles/:roleId.
// Returns 204 No Content on success.
func (h *RoleHandler) UnassignRole(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")

	if err := h.rbacSvc.UnassignRole(c.Request.Context(), userID, roleID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
