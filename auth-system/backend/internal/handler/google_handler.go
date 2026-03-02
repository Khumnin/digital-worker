// internal/handler/google_handler.go
// Sprint 6 — Google OIDC social login handler (US-13).
// API-only per ADR-003: Initiate returns JSON with auth_url; client handles the redirect.
// Callback returns JSON with auth tokens; no server-side HTML or redirects.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// GoogleHandler handles the Google OAuth 2.0 social login flow.
type GoogleHandler struct {
	googleSvc service.GoogleService
	tenantSvc service.TenantService
}

// NewGoogleHandler constructs a GoogleHandler with injected dependencies.
func NewGoogleHandler(googleSvc service.GoogleService, tenantSvc service.TenantService) *GoogleHandler {
	return &GoogleHandler{googleSvc: googleSvc, tenantSvc: tenantSvc}
}

// ── Initiate ──────────────────────────────────────────────────────────────────

type googleInitiateRequest struct {
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// Initiate handles POST /api/v1/auth/oauth/google.
//
// Reads the tenant's Google credentials, generates a state token stored in Redis,
// and returns the Google authorization URL. The client is responsible for
// redirecting the user's browser to that URL (ADR-003: API-only).
//
// Response 200:
//
//	{"auth_url": "https://accounts.google.com/o/oauth2/v2/auth?..."}
//
// Response 400 — missing redirect_uri or ErrGoogleNotConfigured.
func (h *GoogleHandler) Initiate(c *gin.Context) {
	var req googleInitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantIDVal.(string)

	// Load per-tenant config to obtain optional per-tenant Google credentials.
	tenantCfg := domain.TenantConfig{}
	if tenantIDStr != "" {
		if t, err := h.tenantSvc.GetTenant(c.Request.Context(), tenantIDStr); err == nil {
			tenantCfg = t.Config
		}
	}

	result, err := h.googleSvc.InitiateLogin(c.Request.Context(), service.GoogleInitiateInput{
		TenantConfig: tenantCfg,
		RedirectURI:  req.RedirectURI,
		TenantID:     tenantIDStr,
	})
	if err != nil {
		if errors.Is(err, domain.ErrGoogleNotConfigured) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "GOOGLE_NOT_CONFIGURED", "message": err.Error()},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to initiate Google login"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"auth_url": result.AuthURL})
}

// ── Callback ──────────────────────────────────────────────────────────────────

// Callback handles GET /api/v1/auth/oauth/google/callback.
//
// Google redirects the user's browser here with ?code=...&state=... query params.
// tenant_id is taken from a query param because the X-Tenant-ID header is not
// available in a browser redirect from Google.
//
// Optional ?password=... is provided when the client needs to link Google to an
// existing password-based account (ADR-006).
//
// Response 200:
//
//	{"access_token": "...", "refresh_token": "...", "token_type": "Bearer",
//	 "expires_in": 900, "is_new_user": bool}
//
// Response 401 — invalid/expired state (ErrGoogleStateInvalid).
// Response 409 — password required before linking (ErrPasswordRequiredForLinking).
func (h *GoogleHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	tenantIDStr := c.Query("tenant_id")
	password := c.Query("password") // optional — for account linking only

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "INVALID_REQUEST", "message": "code and state query parameters are required"},
		})
		return
	}

	// Build the redirect_uri that was used during InitiateLogin so Google can validate it.
	// Reconstruct it from the request without the query string.
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	redirectURI := scheme + "://" + c.Request.Host + c.Request.URL.Path

	// Load per-tenant Google credentials.
	tenantCfg := domain.TenantConfig{}
	if tenantIDStr != "" {
		if t, err := h.tenantSvc.GetTenant(c.Request.Context(), tenantIDStr); err == nil {
			tenantCfg = t.Config
		}
	}

	result, err := h.googleSvc.HandleCallback(c.Request.Context(), service.GoogleCallbackInput{
		Code:         code,
		State:        state,
		RedirectURI:  redirectURI,
		TenantConfig: tenantCfg,
		TenantID:     tenantIDStr,
		Password:     password,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrGoogleStateInvalid):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_STATE", "message": err.Error()},
			})
		case errors.Is(err, domain.ErrPasswordRequiredForLinking):
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{"code": "PASSWORD_REQUIRED", "message": err.Error()},
			})
		case errors.Is(err, domain.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "the provided password is incorrect"},
			})
		case errors.Is(err, domain.ErrGoogleNotConfigured):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "GOOGLE_NOT_CONFIGURED", "message": err.Error()},
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "GOOGLE_AUTH_ERROR", "message": err.Error()},
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    result.TokenType,
		"expires_in":    result.ExpiresIn,
		"is_new_user":   result.IsNewUser,
	})
}
