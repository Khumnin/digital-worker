// internal/handler/health_handler.go
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler provides a dependency-aware health check endpoint.
// It probes each downstream dependency and returns 503 if any are unreachable,
// enabling load balancers and orchestrators to route traffic away from
// degraded instances.
type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHealthHandler constructs a HealthHandler with the required infrastructure clients.
func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Health handles GET /health.
// Returns 200 with {"status":"ok"} when all dependencies are reachable.
// Returns 503 with {"status":"degraded"} when one or more dependencies fail.
// Each dependency result is reported in the "checks" map for observability.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := gin.H{}
	overallStatus := "ok"
	httpStatus := http.StatusOK

	// PostgreSQL connectivity check.
	if err := h.db.Ping(ctx); err != nil {
		overallStatus = "degraded"
		checks["postgres"] = "error"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["postgres"] = "ok"
	}

	// Redis connectivity check.
	if err := h.redis.Ping(ctx).Err(); err != nil {
		overallStatus = "degraded"
		checks["redis"] = "error"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["redis"] = "ok"
	}

	c.JSON(httpStatus, gin.H{
		"status": overallStatus,
		"checks": checks,
	})
}
