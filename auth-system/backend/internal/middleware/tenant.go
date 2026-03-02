// internal/middleware/tenant.go
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/pkg/apierror"
)

// TenantCache resolves tenant ID/slug to PostgreSQL schema name.
type TenantCache interface {
	GetSchema(ctx context.Context, tenantID string) (string, error)
}

// InMemoryTenantCache is a thread-safe in-memory cache backed by a TenantRepository.
type InMemoryTenantCache struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
	repo  domain.TenantRepository
	ttl   time.Duration
}

type cacheEntry struct {
	schemaName string
	expiresAt  time.Time
}

// NewInMemoryTenantCache creates a new cache.
func NewInMemoryTenantCache(repo domain.TenantRepository, ttl time.Duration) *InMemoryTenantCache {
	return &InMemoryTenantCache{
		cache: make(map[string]cacheEntry),
		repo:  repo,
		ttl:   ttl,
	}
}

// GetSchema returns the PostgreSQL schema name for the given tenant identifier.
func (c *InMemoryTenantCache) GetSchema(ctx context.Context, tenantID string) (string, error) {
	c.mu.RLock()
	entry, ok := c.cache[tenantID]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.schemaName, nil
	}

	tenant, err := c.repo.FindBySlug(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if tenant.Status != domain.TenantStatusActive {
		return "", domain.ErrTenantSuspended
	}

	c.mu.Lock()
	c.cache[tenantID] = cacheEntry{
		schemaName: tenant.SchemaName,
		expiresAt:  time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return tenant.SchemaName, nil
}

// Invalidate removes a tenant entry from the cache.
func (c *InMemoryTenantCache) Invalidate(tenantID string) {
	c.mu.Lock()
	delete(c.cache, tenantID)
	c.mu.Unlock()
}

// RequireTenant resolves tenant schema from X-Tenant-ID header or JWT claims.
func RequireTenant(cache TenantCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string

		if claimsVal, exists := c.Get("jwt_claims"); exists {
			claims := claimsVal.(JWTClaims)
			tenantID = claims.TenantID

			if headerTenantID := c.GetHeader("X-Tenant-ID"); headerTenantID != "" {
				if headerTenantID != tenantID {
					c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
						"TENANT_MISMATCH",
						"The tenant in your token does not match the X-Tenant-ID header.",
						nil, requestID(c),
					))
					return
				}
			}
		} else {
			tenantID = c.GetHeader("X-Tenant-ID")
			if tenantID == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
					"MISSING_TENANT_ID", "X-Tenant-ID header is required.", nil, requestID(c),
				))
				return
			}
		}

		schemaName, err := cache.GetSchema(c.Request.Context(), tenantID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
				"INVALID_TENANT", "Tenant not found or is not active.", nil, requestID(c),
			))
			return
		}

		if !domain.IsValidSchemaName(schemaName) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New(
				"INTERNAL_ERROR", "Invalid tenant configuration detected.", nil, requestID(c),
			))
			return
		}

		c.Set(string(pgdb.CtxKeySchemaName), schemaName)
		c.Set(string(pgdb.CtxKeyTenantID), tenantID)

		// Propagate into the standard context.Context so services can read via ctx.Value.
		ctx := context.WithValue(c.Request.Context(), pgdb.CtxKeySchemaName, schemaName)
		ctx = context.WithValue(ctx, pgdb.CtxKeyTenantID, tenantID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
