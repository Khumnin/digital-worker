// internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/pkg/apierror"
	"tigersoft/auth-system/pkg/jwtutil"
)

// JWTClaims mirrors the claims parsed from a validated JWT.
type JWTClaims struct {
	UserID   string
	TenantID string
	Roles    []string
	JTI      string
}

// RequireAuth validates the JWT Bearer token.
func RequireAuth(jwtVerifier jwtutil.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED", "Authorization header is required.", nil, requestID(c),
			))
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED", "Authorization header must use Bearer scheme.", nil, requestID(c),
			))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED", "Bearer token is empty.", nil, requestID(c),
			))
			return
		}

		claims, err := jwtVerifier.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED", "Token is invalid or has expired.", nil, requestID(c),
			))
			return
		}

		c.Set("jwt_claims", JWTClaims{
			UserID:   claims.Subject,
			TenantID: claims.TenantID,
			Roles:    claims.Roles,
			JTI:      claims.ID,
		})
		c.Set(string(pgdb.CtxKeyUserID), claims.Subject)
		c.Set(string(pgdb.CtxKeyUserRoles), claims.Roles)

		// Propagate into the standard context.Context so services can read via ctx.Value.
		ctx := context.WithValue(c.Request.Context(), pgdb.CtxKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, pgdb.CtxKeyUserRoles, claims.Roles)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireRole aborts with 403 if the authenticated user lacks a required role.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED", "Authentication required.", nil, requestID(c),
			))
			return
		}

		claims := claimsVal.(JWTClaims)
		userRoles := make(map[string]struct{}, len(claims.Roles))
		for _, r := range claims.Roles {
			userRoles[r] = struct{}{}
		}

		for _, required := range roles {
			if _, ok := userRoles[required]; ok {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
			"FORBIDDEN", "You do not have permission to perform this action.", nil, requestID(c),
		))
	}
}

func requestID(c *gin.Context) string {
	if id, ok := c.Get(string(pgdb.CtxKeyRequestID)); ok {
		return id.(string)
	}
	return ""
}
