// internal/router/router.go
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/config"
	"tigersoft/auth-system/internal/handler"
	"tigersoft/auth-system/internal/middleware"
	pginfra "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/pkg/jwtutil"
)

// Dependencies holds all handler and middleware dependencies injected by main.go.
type Dependencies struct {
	Config           *config.Config
	AuthHandler      *handler.AuthHandler
	EmailHandler     *handler.EmailHandler
	PasswordHandler  *handler.PasswordHandler
	SessionHandler   *handler.SessionHandler
	UserHandler      *handler.UserHandler
	AdminHandler     *handler.AdminHandler
	TenantHandler    *handler.TenantHandler
	RoleHandler      *handler.RoleHandler
	OAuthHandler     *handler.OAuthHandler
	GoogleHandler    *handler.GoogleHandler
	AuditHandler     *handler.AuditHandler
	WellKnownHandler *handler.WellKnownHandler
	JWTKeyStore      *jwtutil.KeyStore
	TenantCache      middleware.TenantCache
	RateLimiter      middleware.RateLimiter
}

// New builds and returns the configured Gin engine.
func New(deps Dependencies) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.SecureHeaders())
	r.Use(middleware.CORS(deps.Config.CORS.AllowedOrigins))
	r.Use(gin.Recovery())

	r.GET("/health", healthHandler(deps))
	r.GET("/metrics", gin.WrapH(metricsHandler()))
	r.GET("/.well-known/jwks.json", deps.WellKnownHandler.JWKS)

	v1 := r.Group("/api/v1")

	authMW := middleware.RequireAuth(deps.JWTKeyStore)
	tenantFromHeader := middleware.RequireTenant(deps.TenantCache)
	tenantFromJWT := func() gin.HandlerFunc {
		return middleware.RequireTenant(deps.TenantCache)
	}

	rlLogin := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc: middleware.IPKeyFunc, Limit: deps.Config.RateLimit.LoginIPPerMinute,
		Window: time.Minute, Description: "login:ip",
	})
	rlRegister := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc: middleware.IPKeyFunc, Limit: deps.Config.RateLimit.RegisterIPPerMinute,
		Window: time.Minute, Description: "register:ip",
	})
	rlForgot := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc: middleware.IPKeyFunc, Limit: deps.Config.RateLimit.ForgotPasswordIPPerMinute,
		Window: time.Minute, Description: "forgot:ip",
	})
	rlRefresh := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc: middleware.IPKeyFunc, Limit: deps.Config.RateLimit.TokenRefreshIPPerMinute,
		Window: time.Minute, Description: "refresh:ip",
	})

	auth := v1.Group("/auth", tenantFromHeader)
	{
		auth.POST("/register", rlRegister, deps.AuthHandler.Register)
		auth.POST("/login", rlLogin, deps.AuthHandler.Login)
		auth.POST("/verify-email", deps.EmailHandler.VerifyEmail)
		auth.POST("/resend-verification", deps.EmailHandler.ResendVerification)
		auth.POST("/forgot-password", rlForgot, deps.PasswordHandler.ForgotPassword)
		auth.POST("/reset-password", deps.PasswordHandler.ResetPassword)
		auth.POST("/token/refresh", rlRefresh, deps.SessionHandler.Refresh)
		auth.POST("/oauth/google", deps.GoogleHandler.Initiate)
		auth.GET("/oauth/google/callback", deps.GoogleHandler.Callback)
	}

	authed := v1.Group("", authMW, tenantFromJWT())
	{
		authed.POST("/auth/logout", deps.AuthHandler.Logout)
		authed.POST("/auth/logout/all", deps.AuthHandler.LogoutAll)
		authed.GET("/users/me", deps.UserHandler.GetMe)
		authed.PUT("/users/me", deps.UserHandler.UpdateMe)
	}

	adminRole := middleware.RequireRole("admin")
	admin := v1.Group("/admin", authMW, tenantFromJWT(), adminRole)
	{
		admin.POST("/users/invite", deps.AdminHandler.InviteUser)
		admin.PUT("/users/:id/disable", deps.AdminHandler.DisableUser)
		admin.DELETE("/users/:id", deps.AdminHandler.DeleteUser)
		admin.GET("/users", deps.AdminHandler.ListUsers)
		admin.POST("/roles", deps.RoleHandler.CreateRole)
		admin.GET("/roles", deps.RoleHandler.ListRoles)
		admin.POST("/users/:id/roles", deps.RoleHandler.AssignRole)
		admin.DELETE("/users/:id/roles/:roleId", deps.RoleHandler.UnassignRole)
		admin.GET("/audit-log", deps.AuditHandler.List)
		admin.POST("/oauth/clients", deps.OAuthHandler.RegisterClient)
	}

	superAdminRole := middleware.RequireRole("super_admin")
	superAdmin := v1.Group("/admin", authMW, superAdminRole)
	{
		superAdmin.POST("/tenants", deps.TenantHandler.ProvisionTenant)
		superAdmin.GET("/tenants/:id", deps.TenantHandler.GetTenant)
		superAdmin.GET("/tenants", deps.TenantHandler.ListTenants)
		superAdmin.PUT("/tenants/:id/suspend", deps.TenantHandler.SuspendTenant)
		superAdmin.POST("/tenants/:id/credentials", deps.TenantHandler.GenerateCredentials)
		superAdmin.POST("/tenants/:id/credentials/rotate", deps.TenantHandler.RotateCredentials)
	}

	oauth := v1.Group("/oauth")
	{
		oauth.GET("/authorize", authMW, tenantFromJWT(), deps.OAuthHandler.Authorize)
		oauth.POST("/token", deps.OAuthHandler.Token)
		oauth.POST("/introspect", deps.OAuthHandler.Introspect)
		oauth.POST("/revoke", deps.OAuthHandler.Revoke)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "The requested endpoint does not exist."},
		})
	})

	return r
}

func healthHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth-api",
		})
	}
}

func metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// Ensure pginfra is imported (used for context key types in middleware).
var _ = pginfra.CtxKeySchemaName
