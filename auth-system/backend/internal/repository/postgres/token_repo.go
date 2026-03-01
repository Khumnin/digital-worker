// internal/repository/postgres/token_repo.go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// PostgresTokenRepo implements domain.TokenRepository.
type PostgresTokenRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTokenRepo(pool *pgxpool.Pool) *PostgresTokenRepo {
	return &PostgresTokenRepo{pool: pool}
}

func (r *PostgresTokenRepo) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		// Invalidate any previous unused tokens for this user.
		if _, execErr := conn.Exec(ctx, `
			UPDATE password_reset_tokens SET used = true, used_at = now()
			WHERE user_id = $1 AND used = false
		`, userID); execErr != nil {
			return fmt.Errorf("invalidate previous tokens: %w", execErr)
		}

		_, execErr := conn.Exec(ctx, `
			INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, false, now())
		`, userID, tokenHash, expiresAt)
		return execErr
	})
}

func (r *PostgresTokenRepo) FindPasswordResetToken(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var token *domain.PasswordResetToken
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, user_id, token_hash, expires_at, used, used_at, created_at
			FROM password_reset_tokens
			WHERE token_hash = $1
		`, tokenHash)

		t := &domain.PasswordResetToken{}
		scanErr := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Used, &t.UsedAt, &t.CreatedAt)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrInvalidRefreshToken
			}
			return fmt.Errorf("scan password reset token: %w", scanErr)
		}
		token = t
		return nil
	})

	return token, err
}

func (r *PostgresTokenRepo) MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE password_reset_tokens SET used = true, used_at = now()
			WHERE token_hash = $1
		`, tokenHash)
		return execErr
	})
}

func (r *PostgresTokenRepo) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		// Invalidate any previous unused tokens for this user.
		if _, execErr := conn.Exec(ctx, `
			UPDATE email_verification_tokens SET used = true, used_at = now()
			WHERE user_id = $1 AND used = false
		`, userID); execErr != nil {
			return fmt.Errorf("invalidate previous tokens: %w", execErr)
		}

		_, execErr := conn.Exec(ctx, `
			INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at, used, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, false, now())
		`, userID, tokenHash, expiresAt)
		return execErr
	})
}

func (r *PostgresTokenRepo) FindEmailVerificationToken(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var token *domain.EmailVerificationToken
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, user_id, token_hash, expires_at, used, used_at, created_at
			FROM email_verification_tokens
			WHERE token_hash = $1
		`, tokenHash)

		t := &domain.EmailVerificationToken{}
		scanErr := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Used, &t.UsedAt, &t.CreatedAt)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrInvalidRefreshToken
			}
			return fmt.Errorf("scan email verification token: %w", scanErr)
		}
		token = t
		return nil
	})

	return token, err
}

func (r *PostgresTokenRepo) MarkEmailVerificationTokenUsed(ctx context.Context, tokenHash string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE email_verification_tokens SET used = true, used_at = now()
			WHERE token_hash = $1
		`, tokenHash)
		return execErr
	})
}
