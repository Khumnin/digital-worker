// internal/middleware/request_size.go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize limits the request body size to prevent DoS attacks via large
// payloads. The limit is applied using http.MaxBytesReader which causes
// subsequent reads to return an error once the limit is exceeded.
//
// 64 KB is more than sufficient for all auth API payloads (JSON bodies for
// registration, login, token requests, etc. are all well under 1 KB in
// practice). The generous limit ensures legitimate use cases are not affected.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
