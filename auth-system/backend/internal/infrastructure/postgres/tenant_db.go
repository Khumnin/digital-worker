// internal/infrastructure/postgres/tenant_db.go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
)

func WithTenantSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string, fn func(conn *pgx.Conn) error) error {
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name format: %q — must match tenant_[a-z0-9_]{1,50}", schemaName)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for tenant schema %q: %w", schemaName, err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		return fmt.Errorf("set search_path to %q: %w", schemaName, err)
	}

	return fn(conn.Conn())
}

type ContextKey string

const (
	CtxKeySchemaName ContextKey = "tenant_schema_name"
	CtxKeyTenantID   ContextKey = "tenant_id"
	CtxKeyUserID     ContextKey = "user_id"
	CtxKeyUserRoles  ContextKey = "user_roles"
	CtxKeyRequestID  ContextKey = "request_id"
)

func SchemaFromContext(ctx context.Context) (string, error) {
	schema, ok := ctx.Value(CtxKeySchemaName).(string)
	if !ok || schema == "" {
		return "", fmt.Errorf("tenant schema not found in context: TenantMiddleware must run before this handler")
	}
	return schema, nil
}
