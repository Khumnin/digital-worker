// internal/middleware/rate_limit.go
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/pkg/apierror"
)

// RateLimiter is the interface for distributed rate limit checks.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, windowDuration time.Duration) (bool, int, int, error)
}

// RateLimitConfig defines a rate limit rule.
type RateLimitConfig struct {
	KeyFunc     func(c *gin.Context) string
	Limit       int
	Window      time.Duration
	Description string
}

// RateLimit returns a Gin middleware that enforces the rate limit rule.
func RateLimit(limiter RateLimiter, ruleCfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := ruleCfg.KeyFunc(c)
		if key == "" {
			slog.Warn("rate limit key is empty, blocking request",
				"description", ruleCfg.Description,
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, apierror.New(
				"SERVICE_UNAVAILABLE", "Rate limiting service is temporarily unavailable.", nil, requestID(c),
			))
			return
		}

		allowed, _, retryAfter, err := limiter.Allow(
			c.Request.Context(),
			fmt.Sprintf("rl:%s:%s", ruleCfg.Description, key),
			ruleCfg.Limit,
			ruleCfg.Window,
		)
		if err != nil {
			slog.Error("rate limiter unavailable, blocking request",
				"key", key, "description", ruleCfg.Description, "error", err,
			)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, apierror.New(
				"SERVICE_UNAVAILABLE", "Rate limiting service is temporarily unavailable.", nil, requestID(c),
			))
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(ruleCfg.Limit))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ruleCfg.Window).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, apierror.New(
				"RATE_LIMIT_EXCEEDED", "Too many requests. Please wait before retrying.", nil, requestID(c),
			))
			return
		}

		c.Next()
	}
}

// IPKeyFunc returns the client IP as rate limit key.
func IPKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// UserKeyFunc returns a rate limit key based on the authenticated user + tenant.
func UserKeyFunc(c *gin.Context) string {
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")
	if userID != nil && tenantID != nil {
		return fmt.Sprintf("%v:%v", tenantID, userID)
	}
	return c.ClientIP()
}
