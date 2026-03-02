// internal/middleware/request_id.go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// RequestID ensures every request has a unique X-Request-ID.
// If the client sends X-Request-ID, it is propagated; otherwise a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(string(pgdb.CtxKeyRequestID), id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
