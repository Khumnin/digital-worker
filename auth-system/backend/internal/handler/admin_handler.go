// internal/handler/admin_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
	Email     string `json:"email"      validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
}

// InviteUser handles POST /api/v1/admin/users/invite.
// Creates a new user record and dispatches an invitation email.
func (h *AdminHandler) InviteUser(c *gin.Context) {
	var req inviteUserRequest
	if !bindAndValidate(c, &req) {
		return
	}

	user, err := h.adminSvc.InviteUser(c.Request.Context(), req.Email, req.FirstName, req.LastName)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"status":  string(user.Status),
	})
}

// DisableUser handles PUT /api/v1/admin/users/:id/disable.
// Prevents the user from logging in without permanently deleting their data.
func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")

	if err := h.adminSvc.DisableUser(c.Request.Context(), userID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User has been disabled."})
}

// DeleteUser handles DELETE /api/v1/admin/users/:id.
// Permanently removes the user. Returns 204 No Content on success.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	if err := h.adminSvc.DeleteUser(c.Request.Context(), userID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers handles GET /api/v1/admin/users.
// Supports pagination via limit and offset query parameters.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.adminSvc.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(users))
	for i, u := range users {
		items[i] = gin.H{
			"user_id":    u.ID.String(),
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
			"status":     string(u.Status),
			"created_at": u.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
