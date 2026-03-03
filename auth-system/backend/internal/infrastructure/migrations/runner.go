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
	"tigersoft/auth-system/pkg/crypto"
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

	if err := r.migrateTenantSchema(ctx, tenant.SchemaName, direction, steps); err != nil {
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
		if err := r.migrateTenantSchema(ctx, schemaName, "up", 0); err != nil {
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

// BootstrapPlatform provisions the platform tenant and seeds the super_admin user.
// Every step is idempotent — safe to run on every deploy. The admin password hash
// is refreshed on each run so it stays in sync with the secret value.
func (r *MigrationRunner) BootstrapPlatform(ctx context.Context, adminEmail, adminPassword string) error {
	const (
		platformSlug   = "platform"
		platformName   = "Tigersoft Platform"
		platformSchema = "tenant_platform"
	)

	slog.Info("bootstrapping platform tenant")

	// 1. Insert platform tenant row — no-op if it already exists.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO public.tenants
		    (id, slug, name, schema_name, admin_email, status, config, created_at, updated_at)
		VALUES
		    (gen_random_uuid(), $1, $2, $3, $4, 'active', '{}', now(), now())
		ON CONFLICT (slug) DO NOTHING
	`, platformSlug, platformName, platformSchema, adminEmail)
	if err != nil {
		return fmt.Errorf("insert platform tenant: %w", err)
	}
	slog.Info("platform tenant row ensured")

	// 2. Create schema + run all tenant migrations. The fixed migrateTenantSchema
	//    runs CREATE SCHEMA IF NOT EXISTS first, so this is safe on first deploy.
	if err := r.migrateTenantSchema(ctx, platformSchema, "up", 0); err != nil {
		return fmt.Errorf("migrate platform schema: %w", err)
	}
	slog.Info("platform schema migrated")

	// 3. Hash the admin password (fresh hash on every deploy keeps it in sync
	//    with whatever is in the K8s secret at the time of rollout).
	hash, err := crypto.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash platform admin password: %w", err)
	}

	// 4. Upsert super_admin user.
	_, err = r.pool.Exec(ctx, `
		INSERT INTO tenant_platform.users
		    (id, email, password_hash, status, first_name, last_name,
		     email_verified_at, created_at, updated_at)
		VALUES
		    (gen_random_uuid(), $1, $2, 'active', 'Super', 'Admin', now(), now(), now())
		ON CONFLICT (email) DO UPDATE
		    SET password_hash     = EXCLUDED.password_hash,
		        email_verified_at = COALESCE(users.email_verified_at, now()),
		        updated_at        = now()
	`, adminEmail, hash)
	if err != nil {
		return fmt.Errorf("upsert platform admin user: %w", err)
	}
	slog.Info("platform admin user ensured", "email", adminEmail)

	// 5. Assign super_admin role — no-op if already assigned.
	_, err = r.pool.Exec(ctx, `
		INSERT INTO tenant_platform.user_roles (user_id, role_id, assigned_at)
		SELECT u.id, r.id, now()
		FROM   tenant_platform.users u
		JOIN   tenant_platform.roles r ON r.name = 'super_admin'
		WHERE  u.email = $1
		ON CONFLICT DO NOTHING
	`, adminEmail)
	if err != nil {
		return fmt.Errorf("assign super_admin role: %w", err)
	}
	slog.Info("super_admin role assigned", "email", adminEmail)

	slog.Info("platform bootstrap complete")
	return nil
}

// migrateTenantSchema creates the schema if it doesn't exist, then runs
// golang-migrate against it. The schema creation step is critical: without it,
// search_path=<schema>,public falls back to public.schema_migrations when the
// schema doesn't yet exist, corrupting the global migration version table.
func (r *MigrationRunner) migrateTenantSchema(ctx context.Context, schemaName string, direction string, steps int) error {
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %q", schemaName)
	}

	// schemaName is validated by IsValidSchemaName — safe to interpolate.
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for schema creation: %w", err)
	}
	_, err = conn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schemaName)
	conn.Release()
	if err != nil {
		return fmt.Errorf("create schema %q: %w", schemaName, err)
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
