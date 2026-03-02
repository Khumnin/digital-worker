// internal/infrastructure/migrations/tenant_provisioner.go
package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
)

// TenantProvisioner creates and migrates a new tenant schema.
type TenantProvisioner struct {
	pool           *pgxpool.Pool
	dbURL          string
	tenantMigrPath string
}

// NewTenantProvisioner creates a new TenantProvisioner.
func NewTenantProvisioner(pool *pgxpool.Pool, dbURL, tenantMigrPath string) *TenantProvisioner {
	return &TenantProvisioner{
		pool:           pool,
		dbURL:          dbURL,
		tenantMigrPath: tenantMigrPath,
	}
}

// Provision creates the tenant schema and runs all tenant migrations.
// This is called by TenantService when a new tenant is created.
func (p *TenantProvisioner) Provision(ctx context.Context, schemaName string) error {
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %q", schemaName)
	}

	// Create the schema.
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Use %q-style quoting — but since schemaName is validated, direct interpolation is safe.
	// The regex validation is the security control here.
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	if err != nil {
		return fmt.Errorf("create schema %q: %w", schemaName, err)
	}

	slog.Info("tenant schema created", "schema", schemaName)

	// Run tenant migrations in the new schema.
	dsn := fmt.Sprintf("%s&search_path=%s,public&x-migrations-table=schema_migrations",
		p.dbURL, schemaName)

	runner := &MigrationRunner{
		pool:           p.pool,
		dbURL:          dsn,
		tenantMigrPath: p.tenantMigrPath,
	}

	if err := runner.migrateTenantSchema(schemaName, "up", 0); err != nil {
		return fmt.Errorf("migrate new tenant schema %q: %w", schemaName, err)
	}

	slog.Info("tenant schema migrated", "schema", schemaName)
	return nil
}
