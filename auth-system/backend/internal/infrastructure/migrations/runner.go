// internal/infrastructure/migrations/runner.go
package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
)

// MigrationRunner runs golang-migrate migrations against global and tenant schemas.
type MigrationRunner struct {
	pool           *pgxpool.Pool
	dbURL          string
	globalMigrPath string
	tenantMigrPath string
	tenantRepo     domain.TenantRepository
}

// NewMigrationRunner constructs a MigrationRunner.
func NewMigrationRunner(
	pool *pgxpool.Pool,
	dbURL string,
	globalPath string,
	tenantPath string,
	tenantRepo domain.TenantRepository,
) *MigrationRunner {
	return &MigrationRunner{
		pool:           pool,
		dbURL:          dbURL,
		globalMigrPath: globalPath,
		tenantMigrPath: tenantPath,
		tenantRepo:     tenantRepo,
	}
}

// RunGlobal applies migrations to the public (global) schema.
func (r *MigrationRunner) RunGlobal(ctx context.Context, direction string) error {
	m, err := migrate.New(r.globalMigrPath, r.dbURL)
	if err != nil {
		return fmt.Errorf("create global migrator: %w", err)
	}
	defer m.Close()
	return runMigration(m, direction, 0)
}

// RunTenant applies migrations to a single named tenant schema.
func (r *MigrationRunner) RunTenant(ctx context.Context, tenantSlug string, direction string, steps int) error {
	tenant, err := r.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil {
		return fmt.Errorf("find tenant %q: %w", tenantSlug, err)
	}

	if err := r.migrateTenantSchema(tenant.SchemaName, direction, steps); err != nil {
		return fmt.Errorf("migrate tenant %q (schema %q): %w", tenantSlug, tenant.SchemaName, err)
	}
	return nil
}

// RunAllTenants applies pending UP migrations to all active tenant schemas.
func (r *MigrationRunner) RunAllTenants(ctx context.Context) error {
	schemas, err := r.tenantRepo.ListActiveSchemaNames(ctx)
	if err != nil {
		return fmt.Errorf("list active tenant schemas: %w", err)
	}

	var failedSchemas []string
	for _, schemaName := range schemas {
		if err := r.migrateTenantSchema(schemaName, "up", 0); err != nil {
			slog.Error("tenant migration failed", "schema", schemaName, "error", err)
			failedSchemas = append(failedSchemas, schemaName)
		} else {
			slog.Info("tenant migration applied", "schema", schemaName)
		}
	}

	if len(failedSchemas) > 0 {
		return fmt.Errorf("migrations failed for %d schema(s): %v", len(failedSchemas), failedSchemas)
	}
	return nil
}

func (r *MigrationRunner) migrateTenantSchema(schemaName string, direction string, steps int) error {
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %q", schemaName)
	}

	dsn := fmt.Sprintf("%s&search_path=%s,public&x-migrations-table=schema_migrations",
		r.dbURL, schemaName)

	m, err := migrate.New(r.tenantMigrPath, dsn)
	if err != nil {
		return fmt.Errorf("create migrator for schema %q: %w", schemaName, err)
	}
	defer m.Close()

	return runMigration(m, direction, steps)
}

func runMigration(m *migrate.Migrate, direction string, steps int) error {
	var err error
	switch direction {
	case "up":
		err = m.Up()
	case "down":
		if steps > 0 {
			err = m.Steps(-steps)
		} else {
			err = m.Down()
		}
	default:
		return fmt.Errorf("unknown migration direction: %q", direction)
	}

	if err == migrate.ErrNoChange {
		slog.Info("no new migrations to apply")
		return nil
	}
	return err
}
