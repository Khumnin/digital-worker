// internal/repository/postgres/session_repo.go
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

// PostgresSessionRepo implements domain.SessionRepository.
type PostgresSessionRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepo(pool *pgxpool.Pool) *PostgresSessionRepo {
	return &PostgresSessionRepo{pool: pool}
}

func (r *PostgresSessionRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var session *domain.Session
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, user_id, refresh_token_hash, family_id, ip_address, user_agent,
			       issued_at, expires_at, last_used_at, revoked_at, is_revoked
			FROM sessions
			WHERE refresh_token_hash = $1
		`, tokenHash)

		s, scanErr := scanSession(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrSessionNotFound
			}
			return fmt.Errorf("scan session: %w", scanErr)
		}
		session = s
		return nil
	})

	return session, err
}

func (r *PostgresSessionRepo) Create(ctx context.Context, session *domain.Session) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			INSERT INTO sessions (id, user_id, refresh_token_hash, family_id, ip_address, user_agent,
			                      issued_at, expires_at, last_used_at, is_revoked)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			session.ID, session.UserID, session.RefreshTokenHash, session.FamilyID,
			session.IPAddress, session.UserAgent, session.IssuedAt, session.ExpiresAt,
			session.LastUsedAt, session.IsRevoked,
		)
		return execErr
	})
}

func (r *PostgresSessionRepo) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		now := time.Now()
		_, execErr := conn.Exec(ctx, `
			UPDATE sessions SET is_revoked = true, revoked_at = $2
			WHERE refresh_token_hash = $1
		`, tokenHash, now)
		return execErr
	})
}

func (r *PostgresSessionRepo) RevokeByFamilyID(ctx context.Context, familyID uuid.UUID) (int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var count int
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		now := time.Now()
		tag, execErr := conn.Exec(ctx, `
			UPDATE sessions SET is_revoked = true, revoked_at = $2
			WHERE family_id = $1 AND is_revoked = false
		`, familyID, now)
		count = int(tag.RowsAffected())
		return execErr
	})

	return count, err
}

func (r *PostgresSessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var count int
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		now := time.Now()
		tag, execErr := conn.Exec(ctx, `
			UPDATE sessions SET is_revoked = true, revoked_at = $2
			WHERE user_id = $1 AND is_revoked = false
		`, userID, now)
		count = int(tag.RowsAffected())
		return execErr
	})

	return count, err
}

func (r *PostgresSessionRepo) CountActiveForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var count int
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		return conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM sessions
			WHERE user_id = $1 AND is_revoked = false AND expires_at > now()
		`, userID).Scan(&count)
	})

	return count, err
}

func scanSession(row pgx.Row) (*domain.Session, error) {
	s := &domain.Session{}
	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.FamilyID,
		&s.IPAddress, &s.UserAgent, &s.IssuedAt, &s.ExpiresAt,
		&s.LastUsedAt, &s.RevokedAt, &s.IsRevoked,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}
