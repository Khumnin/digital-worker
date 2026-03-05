// internal/handler/admin_handler.go
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tigersoft/auth-system/internal/middleware"
	"tigersoft/auth-system/internal/service"
)

// AdminHandler handles tenant-scoped user administration endpoints.
type AdminHandler struct {
	adminSvc service.AdminService
}

// NewAdminHandler constructs an AdminHandler with its required service dependency.
func NewAdminHandler(svc service.AdminService) *AdminHandler {
	return &AdminHandler{adminSvc: svc}
}

type inviteUserRequest struct {
	Email       string `json:"email"        validate:"required,email"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=200"`
	InitialRole string `json:"initial_role"` // optional; defaults to "user" if omitted
}

// InviteUser handles POST /api/v1/admin/users/invite.
// Creates a new user record and dispatches an invitation email.
// Returns a full user object with status="pending", empty roles.
// The request body must include "email" and "display_name".
func (h *AdminHandler) InviteUser(c *gin.Context) {
	var req inviteUserRequest
	if !bindAndValidate(c, &req) {
		return
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	// Pass the full display_name as firstName; lastName is left empty so that
	// domain.User.FullName() returns the display_name verbatim.
	user, err := h.adminSvc.InviteUser(c.Request.Context(), req.Email, req.DisplayName, "", req.InitialRole, claims.UserID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           user.ID.String(),
		"email":        user.Email,
		"display_name": user.FullName(),
		"status":       normalizeUserStatus(string(user.Status)),
		"system_roles": []string{},
		"module_roles": map[string][]string{},
		"tenant_id":    claims.TenantID,
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	})
}

// ResendInvite handles POST /api/v1/admin/users/:id/resend-invite.
// Re-sends the invitation email for a user that has not yet accepted their invite.
func (h *AdminHandler) ResendInvite(c *gin.Context) {
	userID := c.Param("id")

	if err := h.adminSvc.ResendInvite(c.Request.Context(), userID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation re-sent successfully."})
}

// DisableUser handles POST /api/v1/admin/users/:id/disable.
// Prevents the user from logging in without permanently deleting their data.
// Guards:
//  1. An admin cannot suspend themselves.
//  2. A non-super_admin cannot suspend a super_admin.
func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	// Guard 1: cannot suspend yourself.
	if claims.UserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]string{
			"code":    "CANNOT_SUSPEND_SELF",
			"message": "You cannot suspend your own account.",
		}})
		return
	}

	// Guard 2: only super_admin can suspend another super_admin.
	actorIsSuperAdmin := false
	for _, r := range claims.Roles {
		if r == "super_admin" {
			actorIsSuperAdmin = true
			break
		}
	}
	if !actorIsSuperAdmin {
		target, err := h.adminSvc.GetUser(c.Request.Context(), userID)
		if err == nil {
			for _, r := range target.SystemRoles {
				if r == "super_admin" {
					c.JSON(http.StatusForbidden, gin.H{"error": map[string]string{
						"code":    "CANNOT_SUSPEND_SUPER_ADMIN",
						"message": "Only a super admin can suspend another super admin.",
					}})
					return
				}
			}
		}
	}

	if err := h.adminSvc.DisableUser(c.Request.Context(), userID, claims.UserID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User has been disabled."})
}

// EnableUser handles POST /api/v1/admin/users/:id/enable.
// Restores a disabled user account to active status.
func (h *AdminHandler) EnableUser(c *gin.Context) {
	userID := c.Param("id")

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	if err := h.adminSvc.EnableUser(c.Request.Context(), userID, claims.UserID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User has been enabled."})
}

// normalizeUserStatus maps internal DB status values to the API contract values.
// "unverified" → "pending"   (user invited but has not set a password yet)
// "disabled"   → "inactive"  (admin-disabled account)
// All other values pass through unchanged.
func normalizeUserStatus(status string) string {
	switch status {
	case "unverified":
		return "pending"
	case "disabled":
		return "inactive"
	default:
		return status
	}
}

// denormalizeUserStatus maps API status values back to internal DB values for filtering.
func denormalizeUserStatus(status string) string {
	switch status {
	case "pending":
		return "unverified"
	case "inactive":
		return "disabled"
	default:
		return status
	}
}

// DeleteUser handles DELETE /api/v1/admin/users/:id.
// Performs a full GDPR erasure: anonymizes PII, revokes sessions, removes
// MFA codes, social links, and OAuth codes. Returns 204 No Content on success.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Extract the requesting admin's identity for the audit trail.
	requestedBy := uuid.Nil
	if claimsVal, exists := c.Get("jwt_claims"); exists {
		claims := claimsVal.(middleware.JWTClaims)
		if parsed, parseErr := uuid.Parse(claims.UserID); parseErr == nil {
			requestedBy = parsed
		}
	}

	if err := h.adminSvc.EraseUser(c.Request.Context(), userID, requestedBy); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers handles GET /api/v1/admin/users.
// Supports pagination via page and page_size query parameters.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Optional status filter. Validate the raw API value before accepting it.
	// Accepted values: "pending", "inactive", "active", "all" (empty string → no filter).
	rawStatus := c.Query("status")
	if rawStatus != "" && rawStatus != "pending" && rawStatus != "inactive" && rawStatus != "active" && rawStatus != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status filter; accepted values: pending, inactive, active, all"})
		return
	}
	if rawStatus == "all" {
		rawStatus = ""
	}

	// Denormalize API status values to internal DB values before querying.
	statusFilter := denormalizeUserStatus(rawStatus)

	users, total, err := h.adminSvc.ListUsers(c.Request.Context(), pageSize, offset, statusFilter)
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

// GetUser handles GET /api/v1/admin/users/:id.
// Returns a full user object including resolved system and module roles.
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	uwr, err := h.adminSvc.GetUser(c.Request.Context(), userID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildUserResponse(uwr, claims.TenantID))
}

type replaceUserRolesRequest struct {
	SystemRoles []string            `json:"system_roles"`
	ModuleRoles map[string][]string `json:"module_roles"`
}

// ReplaceUserRoles handles PUT /api/v1/admin/users/:id/roles.
// Atomically replaces all role assignments for a user.
func (h *AdminHandler) ReplaceUserRoles(c *gin.Context) {
	userID := c.Param("id")

	var req replaceUserRolesRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if req.SystemRoles == nil {
		req.SystemRoles = []string{}
	}
	if req.ModuleRoles == nil {
		req.ModuleRoles = map[string][]string{}
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims := claimsVal.(middleware.JWTClaims)

	uwr, err := h.adminSvc.ReplaceUserRoles(c.Request.Context(), userID, req.SystemRoles, req.ModuleRoles, claims.UserID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildUserResponse(uwr, claims.TenantID))
}

// buildUserResponse constructs the standard user response object (BE-009 shape).
func buildUserResponse(uwr *service.UserWithRoles, tenantID string) gin.H {
	systemRoles := uwr.SystemRoles
	if systemRoles == nil {
		systemRoles = []string{}
	}
	moduleRoles := uwr.ModuleRoles
	if moduleRoles == nil {
		moduleRoles = map[string][]string{}
	}

	return gin.H{
		"id":           uwr.User.ID.String(),
		"email":        uwr.User.Email,
		"display_name": uwr.User.FullName(),
		"status":       normalizeUserStatus(string(uwr.User.Status)),
		"system_roles": systemRoles,
		"module_roles": moduleRoles,
		"tenant_id":    tenantID,
		"created_at":   uwr.User.CreatedAt,
		"updated_at":   uwr.User.UpdatedAt,
	}
}
