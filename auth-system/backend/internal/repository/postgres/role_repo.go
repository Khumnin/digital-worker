// internal/repository/postgres/role_repo.go
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// PostgresRoleRepo implements domain.RoleRepository.
type PostgresRoleRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRoleRepo(pool *pgxpool.Pool) *PostgresRoleRepo {
	return &PostgresRoleRepo{pool: pool}
}

func (r *PostgresRoleRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var role *domain.Role
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, name, description, module, is_system, created_at FROM roles WHERE id = $1
		`, id)
		ro, scanErr := scanRole(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrRoleNotFound
			}
			return scanErr
		}
		role = ro
		return nil
	})
	return role, err
}

func (r *PostgresRoleRepo) FindByName(ctx context.Context, name string) (*domain.Role, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var role *domain.Role
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, name, description, module, is_system, created_at FROM roles WHERE name = $1
		`, name)
		ro, scanErr := scanRole(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrRoleNotFound
			}
			return scanErr
		}
		role = ro
		return nil
	})
	return role, err
}

func (r *PostgresRoleRepo) ListAll(ctx context.Context) ([]*domain.Role, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var roles []*domain.Role
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		rows, queryErr := conn.Query(ctx, `SELECT id, name, description, module, is_system, created_at FROM roles ORDER BY name`)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()
		for rows.Next() {
			ro := &domain.Role{}
			if scanErr := rows.Scan(&ro.ID, &ro.Name, &ro.Description, &ro.Module, &ro.IsSystem, &ro.CreatedAt); scanErr != nil {
				return scanErr
			}
			roles = append(roles, ro)
		}
		return rows.Err()
	})
	return roles, err
}

func (r *PostgresRoleRepo) Create(ctx context.Context, name, description string) (*domain.Role, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var role *domain.Role
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			INSERT INTO roles (id, name, description, is_system, created_at)
			VALUES (gen_random_uuid(), $1, $2, false, now())
			RETURNING id, name, description, module, is_system, created_at
		`, name, description)
		ro, scanErr := scanRole(row)
		if scanErr != nil {
			if isUniqueViolation(scanErr) {
				return domain.ErrRoleAlreadyExists
			}
			return scanErr
		}
		role = ro
		return nil
	})
	return role, err
}

func (r *PostgresRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tag, execErr := conn.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrRoleNotFound
		}
		return nil
	})
}

func (r *PostgresRoleRepo) IsAssignedToAnyUser(ctx context.Context, id uuid.UUID) (bool, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return false, err
	}

	var assigned bool
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles WHERE role_id = $1)`, id)
		return row.Scan(&assigned)
	})
	return assigned, err
}

func (r *PostgresRoleRepo) AssignToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, assigned_by, assigned_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT DO NOTHING
		`, userID, roleID, assignedBy)
		return execErr
	})
}

func (r *PostgresRoleRepo) UnassignFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2
		`, userID, roleID)
		return execErr
	})
}

func (r *PostgresRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*domain.Role, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var roles []*domain.Role
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		rows, queryErr := conn.Query(ctx, `
			SELECT r.id, r.name, r.description, r.module, r.is_system, r.created_at
			FROM roles r
			JOIN user_roles ur ON ur.role_id = r.id
			WHERE ur.user_id = $1
		`, userID)
		if queryErr != nil {
			return fmt.Errorf("get user roles: %w", queryErr)
		}
		defer rows.Close()
		for rows.Next() {
			ro := &domain.Role{}
			if scanErr := rows.Scan(&ro.ID, &ro.Name, &ro.Description, &ro.Module, &ro.IsSystem, &ro.CreatedAt); scanErr != nil {
				return scanErr
			}
			roles = append(roles, ro)
		}
		return rows.Err()
	})
	return roles, err
}

// ReplaceUserRoles atomically removes all existing role assignments for a user
// and inserts the new set. Runs inside a single database transaction so the
// operation is all-or-nothing.
func (r *PostgresRoleRepo) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tx, txErr := conn.Begin(ctx)
		if txErr != nil {
			return fmt.Errorf("begin transaction: %w", txErr)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		// Delete all existing role assignments for this user.
		if _, delErr := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); delErr != nil {
			return fmt.Errorf("delete user roles: %w", delErr)
		}

		// Insert the new roles.
		for _, roleID := range roleIDs {
			if _, insErr := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id, assigned_by, assigned_at)
				VALUES ($1, $2, $1, now())
				ON CONFLICT DO NOTHING
			`, userID, roleID); insErr != nil {
				return fmt.Errorf("insert user role %s: %w", roleID, insErr)
			}
		}

		return tx.Commit(ctx)
	})
}

func scanRole(row pgx.Row) (*domain.Role, error) {
	ro := &domain.Role{}
	err := row.Scan(&ro.ID, &ro.Name, &ro.Description, &ro.Module, &ro.IsSystem, &ro.CreatedAt)
	if err != nil {
		return nil, err
	}
	return ro, nil
}
