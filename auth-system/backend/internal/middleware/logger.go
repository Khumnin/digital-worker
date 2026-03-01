// internal/middleware/logger.go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// StructuredLogger logs each HTTP request as a structured slog event.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		reqID := ""
		if v, ok := c.Get(string(pgdb.CtxKeyRequestID)); ok {
			reqID, _ = v.(string)
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "http_request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"request_id", reqID,
			"errors", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}
