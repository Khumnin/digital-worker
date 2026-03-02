// internal/repository/postgres/user_repo.go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// PostgresUserRepo implements domain.UserRepository.
type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

func (r *PostgresUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users WHERE id = $1 AND deleted_at IS NULL
		`, id)
		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("scan user by id: %w", scanErr)
		}
		user = u
		return nil
	})
	return user, err
}

func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users WHERE email = $1 AND deleted_at IS NULL
		`, email)
		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("scan user by email: %w", scanErr)
		}
		user = u
		return nil
	})
	return user, err
}

func (r *PostgresUserRepo) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			INSERT INTO users (id, email, password_hash, status, first_name, last_name,
			                   mfa_enabled, failed_login_count, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, false, 0, now(), now())
			RETURNING id, email, password_hash, status, first_name, last_name,
			          mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			          email_verified_at, last_login_at, created_at, updated_at, deleted_at
		`, input.Email, input.PasswordHash, string(input.Status), input.FirstName, input.LastName)
		u, scanErr := scanUser(row)
		if scanErr != nil {
			if isUniqueViolation(scanErr) {
				return domain.ErrEmailAlreadyExists
			}
			return fmt.Errorf("insert user: %w", scanErr)
		}
		user = u
		return nil
	})
	return user, err
}

func (r *PostgresUserRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		setClauses := []string{"updated_at = now()"}
		args := []interface{}{}
		argIdx := 1

		if input.FirstName != nil {
			setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIdx))
			args = append(args, *input.FirstName)
			argIdx++
		}
		if input.LastName != nil {
			setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIdx))
			args = append(args, *input.LastName)
			argIdx++
		}
		if input.PasswordHash != nil {
			setClauses = append(setClauses, fmt.Sprintf("password_hash = $%d", argIdx))
			args = append(args, *input.PasswordHash)
			argIdx++
		}
		if input.Status != nil {
			setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, string(*input.Status))
			argIdx++
		}
		if input.MFAEnabled != nil {
			setClauses = append(setClauses, fmt.Sprintf("mfa_enabled = $%d", argIdx))
			args = append(args, *input.MFAEnabled)
			argIdx++
		}
		if input.MFATOTPSecret != nil {
			setClauses = append(setClauses, fmt.Sprintf("mfa_totp_secret = $%d", argIdx))
			args = append(args, *input.MFATOTPSecret)
			argIdx++
		}
		if input.FailedLoginCount != nil {
			setClauses = append(setClauses, fmt.Sprintf("failed_login_count = $%d", argIdx))
			args = append(args, *input.FailedLoginCount)
			argIdx++
		}
		if input.LockedUntil != nil {
			setClauses = append(setClauses, fmt.Sprintf("locked_until = $%d", argIdx))
			args = append(args, *input.LockedUntil)
			argIdx++
		}
		if input.EmailVerifiedAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("email_verified_at = $%d", argIdx))
			args = append(args, *input.EmailVerifiedAt)
			argIdx++
		}
		if input.LastLoginAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("last_login_at = $%d", argIdx))
			args = append(args, *input.LastLoginAt)
			argIdx++
		}
		if input.DeletedAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("deleted_at = $%d", argIdx))
			args = append(args, *input.DeletedAt)
			argIdx++
		}

		args = append(args, id)
		query := fmt.Sprintf(`
			UPDATE users SET %s
			WHERE id = $%d AND deleted_at IS NULL
			RETURNING id, email, password_hash, status, first_name, last_name,
			          mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			          email_verified_at, last_login_at, created_at, updated_at, deleted_at
		`, strings.Join(setClauses, ", "), argIdx)

		row := conn.QueryRow(ctx, query, args...)
		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("update user: %w", scanErr)
		}
		user = u
		return nil
	})
	return user, err
}

func (r *PostgresUserRepo) IncrementFailedLoginCount(ctx context.Context, id uuid.UUID) (int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		return conn.QueryRow(ctx, `
			UPDATE users SET failed_login_count = failed_login_count + 1, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL RETURNING failed_login_count
		`, id).Scan(&newCount)
	})
	return newCount, err
}

func (r *PostgresUserRepo) ResetFailedLoginCount(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users SET failed_login_count = 0, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id)
		return execErr
	})
}

func (r *PostgresUserRepo) SetLockedUntil(ctx context.Context, id uuid.UUID, until time.Time) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users SET locked_until = $2, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id, until)
		return execErr
	})
}

func (r *PostgresUserRepo) ListByTenant(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	var users []*domain.User
	var total int

	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		if countErr := conn.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&total); countErr != nil {
			return fmt.Errorf("count users: %w", countErr)
		}

		rows, queryErr := conn.Query(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
		if queryErr != nil {
			return fmt.Errorf("query users: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			u, scanErr := scanUserFromRows(rows)
			if scanErr != nil {
				return fmt.Errorf("scan user row: %w", scanErr)
			}
			users = append(users, u)
		}
		return rows.Err()
	})

	return users, total, err
}

func (r *PostgresUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tag, execErr := conn.Exec(ctx, `
			UPDATE users SET deleted_at = now(), status = 'deleted', updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrUserNotFound
		}
		return nil
	})
}

func (r *PostgresUserRepo) AnonymizePII(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users SET
			    email        = 'deleted-' || id::text || '@gdpr.tombstone',
			    first_name   = '[deleted]',
			    last_name    = '[deleted]',
			    password_hash = '',
			    mfa_totp_secret = NULL,
			    updated_at   = now()
			WHERE id = $1
		`, id)
		return execErr
	})
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Status, &u.FirstName, &u.LastName,
		&u.MFAEnabled, &u.MFATOTPSecret, &u.FailedLoginCount, &u.LockedUntil,
		&u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func scanUserFromRows(rows pgx.Rows) (*domain.User, error) {
	u := &domain.User{}
	err := rows.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Status, &u.FirstName, &u.LastName,
		&u.MFAEnabled, &u.MFATOTPSecret, &u.FailedLoginCount, &u.LockedUntil,
		&u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

