// internal/repository/postgres/mfa_repo.go
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

// PostgresMFARepo implements domain.MFARepository using pgx.
type PostgresMFARepo struct {
	pool *pgxpool.Pool
}

// NewPostgresMFARepo constructs a PostgresMFARepo.
func NewPostgresMFARepo(pool *pgxpool.Pool) *PostgresMFARepo {
	return &PostgresMFARepo{pool: pool}
}

// CreateBackupCodes replaces all existing backup codes for a user with the
// provided set of hashed codes, executed atomically within a transaction.
func (r *PostgresMFARepo) CreateBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Delete all existing codes for this user first.
		if _, err := tx.Exec(ctx,
			`DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID,
		); err != nil {
			return fmt.Errorf("delete existing backup codes: %w", err)
		}

		// Batch insert the new hashed codes.
		for _, hash := range codeHashes {
			if _, err := tx.Exec(ctx,
				`INSERT INTO mfa_backup_codes (id, user_id, code_hash, used, created_at)
				 VALUES (gen_random_uuid(), $1, $2, false, now())`,
				userID, hash,
			); err != nil {
				return fmt.Errorf("insert backup code: %w", err)
			}
		}

		return tx.Commit(ctx)
	})
}

// ConsumeBackupCode finds an unused backup code matching the provided hash and
// marks it as used. Returns ErrBackupCodeInvalid if no match is found.
func (r *PostgresMFARepo) ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tag, execErr := conn.Exec(ctx, `
			UPDATE mfa_backup_codes
			SET used = true, used_at = now()
			WHERE user_id = $1 AND code_hash = $2 AND used = false
		`, userID, codeHash)
		if execErr != nil {
			return fmt.Errorf("consume backup code: %w", execErr)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrBackupCodeInvalid
		}
		return nil
	})
}

// DeleteAllForUser removes all backup codes for the given user. Called when
// MFA is disabled.
func (r *PostgresMFARepo) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx,
			`DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID,
		)
		if execErr != nil && !errors.Is(execErr, pgx.ErrNoRows) {
			return fmt.Errorf("delete backup codes: %w", execErr)
		}
		return nil
	})
}
