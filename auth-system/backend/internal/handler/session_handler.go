// internal/handler/session_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/service"
)

// SessionHandler handles token refresh endpoints.
type SessionHandler struct {
	sessionSvc service.SessionService
}

// NewSessionHandler constructs a SessionHandler with its required service dependency.
func NewSessionHandler(svc service.SessionService) *SessionHandler {
	return &SessionHandler{sessionSvc: svc}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=1"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Refresh handles POST /api/v1/auth/token/refresh.
// Rotates the refresh token and issues a new access token.
func (h *SessionHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.sessionSvc.Refresh(
		c.Request.Context(),
		req.RefreshToken,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, refreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
	})
}
