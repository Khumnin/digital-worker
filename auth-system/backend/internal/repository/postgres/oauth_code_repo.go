// internal/repository/postgres/oauth_code_repo.go
package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

type postgresAuthCodeRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresAuthCodeRepo(pool *pgxpool.Pool) domain.AuthorizationCodeRepository {
	return &postgresAuthCodeRepo{pool: pool}
}

func (r *postgresAuthCodeRepo) Create(ctx context.Context, code *domain.AuthorizationCode) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO oauth_authorization_codes
				(id, code_hash, client_id, user_id, redirect_uri, scopes,
				 code_challenge, code_challenge_method, expires_at, used, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,false,now())`,
			code.ID, code.CodeHash, code.ClientID, code.UserID, code.RedirectURI,
			code.Scopes, code.CodeChallenge, code.CodeChallengeMethod, code.ExpiresAt,
		)
		return err
	})
}

func (r *postgresAuthCodeRepo) FindByCodeHash(ctx context.Context, codeHash string) (*domain.AuthorizationCode, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var code domain.AuthorizationCode
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, code_hash, client_id, user_id, redirect_uri, scopes,
			       code_challenge, code_challenge_method, expires_at, used, created_at
			FROM oauth_authorization_codes
			WHERE code_hash = $1`, codeHash)
		return row.Scan(
			&code.ID, &code.CodeHash, &code.ClientID, &code.UserID,
			&code.RedirectURI, &code.Scopes,
			&code.CodeChallenge, &code.CodeChallengeMethod,
			&code.ExpiresAt, &code.Used, &code.CreatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAuthCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *postgresAuthCodeRepo) MarkUsed(ctx context.Context, codeHash string) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, err := conn.Exec(ctx,
			`UPDATE oauth_authorization_codes SET used = true WHERE code_hash = $1`, codeHash)
		return err
	})
}

// DeleteByUserID removes all authorization codes issued to the given user.
// Called on password change and GDPR erasure to invalidate outstanding codes.
func (r *postgresAuthCodeRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx,
			`DELETE FROM oauth_authorization_codes WHERE user_id = $1`, userID)
		return execErr
	})
}
