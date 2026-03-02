// internal/repository/postgres/tenant_repo.go
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
)

// PostgresTenantRepo implements domain.TenantRepository.
// Operates on the global public schema — no tenant schema routing needed.
type PostgresTenantRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTenantRepo(pool *pgxpool.Pool) *PostgresTenantRepo {
	return &PostgresTenantRepo{pool: pool}
}

func (r *PostgresTenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, schema_name, admin_email, status, config, created_at, updated_at, deleted_at
		FROM tenants WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return scanTenant(row)
}

func (r *PostgresTenantRepo) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, schema_name, admin_email, status, config, created_at, updated_at, deleted_at
		FROM tenants WHERE slug = $1 AND deleted_at IS NULL
	`, slug)
	return scanTenant(row)
}

func (r *PostgresTenantRepo) FindBySchemaName(ctx context.Context, schemaName string) (*domain.Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, schema_name, admin_email, status, config, created_at, updated_at, deleted_at
		FROM tenants WHERE schema_name = $1 AND deleted_at IS NULL
	`, schemaName)
	return scanTenant(row)
}

func (r *PostgresTenantRepo) Create(ctx context.Context, input domain.CreateTenantInput) (*domain.Tenant, error) {
	schemaName := domain.SlugToSchemaName(input.Slug)
	configJSON, err := json.Marshal(input.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal tenant config: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO tenants (id, slug, name, schema_name, admin_email, status, config, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'active', $5, now(), now())
		RETURNING id, slug, name, schema_name, admin_email, status, config, created_at, updated_at, deleted_at
	`, input.Slug, input.Name, schemaName, input.AdminEmail, configJSON)

	t, err := scanTenant(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrTenantAlreadyExists
		}
		return nil, err
	}
	return t, nil
}

func (r *PostgresTenantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TenantStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tenants SET status = $2, updated_at = now() WHERE id = $1
	`, id, string(status))
	return err
}

func (r *PostgresTenantRepo) UpdateConfig(ctx context.Context, id uuid.UUID, config domain.TenantConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal tenant config: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE tenants SET config = $2, updated_at = now() WHERE id = $1
	`, id, configJSON)
	return err
}

func (r *PostgresTenantRepo) ListActiveSchemaNames(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT schema_name FROM tenants WHERE status = 'active' AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("query active schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, scanErr
		}
		schemas = append(schemas, name)
	}
	return schemas, rows.Err()
}

func (r *PostgresTenantRepo) ListAll(ctx context.Context, limit, offset int) ([]*domain.Tenant, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, schema_name, admin_email, status, config, created_at, updated_at, deleted_at
		FROM tenants WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		t, scanErr := scanTenantFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		tenants = append(tenants, t)
	}
	return tenants, total, rows.Err()
}

func scanTenant(row pgx.Row) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var configJSON []byte
	var statusStr string
	err := row.Scan(
		&t.ID, &t.Slug, &t.Name, &t.SchemaName, &t.AdminEmail,
		&statusStr, &configJSON, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("scan tenant: %w", err)
	}
	t.Status = domain.TenantStatus(statusStr)
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &t.Config)
	}
	return t, nil
}

func scanTenantFromRows(rows pgx.Rows) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	var configJSON []byte
	var statusStr string
	err := rows.Scan(
		&t.ID, &t.Slug, &t.Name, &t.SchemaName, &t.AdminEmail,
		&statusStr, &configJSON, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan tenant row: %w", err)
	}
	t.Status = domain.TenantStatus(statusStr)
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &t.Config)
	}
	return t, nil
}

// isUniqueViolation returns true when err is a PostgreSQL unique_violation (SQLSTATE 23505).
// Used by tenant_repo and role_repo to detect duplicate key errors without string matching.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
