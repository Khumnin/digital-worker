// internal/repository/postgres/social_account_repo.go
// Sprint 6 — SocialAccountRepository backed by the oauth_social_accounts table.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

type postgresSocialAccountRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresSocialAccountRepo constructs a SocialAccountRepository backed by PostgreSQL.
func NewPostgresSocialAccountRepo(pool *pgxpool.Pool) domain.SocialAccountRepository {
	return &postgresSocialAccountRepo{pool: pool}
}

// FindByProvider returns the social account for (provider, provider_user_id).
// Returns domain.ErrSocialAccountNotFound when no row exists.
func (r *postgresSocialAccountRepo) FindByProvider(ctx context.Context, provider, providerUserID string) (*domain.SocialAccount, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var acct domain.SocialAccount
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, user_id, provider, provider_user_id, email,
			       access_token, refresh_token, expires_at, created_at, updated_at
			FROM oauth_social_accounts
			WHERE provider = $1 AND provider_user_id = $2
		`, provider, providerUserID)
		return row.Scan(
			&acct.ID, &acct.UserID, &acct.Provider, &acct.ProviderUserID, &acct.Email,
			&acct.AccessToken, &acct.RefreshToken, &acct.ExpiresAt,
			&acct.CreatedAt, &acct.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSocialAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

// Create inserts a new row into oauth_social_accounts.
func (r *postgresSocialAccountRepo) Create(ctx context.Context, acct *domain.SocialAccount) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			INSERT INTO oauth_social_accounts
				(id, user_id, provider, provider_user_id, email,
				 access_token, refresh_token, expires_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		`,
			acct.ID, acct.UserID, acct.Provider, acct.ProviderUserID, acct.Email,
			acct.AccessToken, acct.RefreshToken, acct.ExpiresAt,
		)
		return execErr
	})
}

// UpdateTokens refreshes the access/refresh tokens and expiry timestamp for an existing account.
func (r *postgresSocialAccountRepo) UpdateTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiresAt *time.Time) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE oauth_social_accounts
			SET access_token = $1, refresh_token = $2, expires_at = $3, updated_at = now()
			WHERE id = $4
		`, accessToken, refreshToken, expiresAt, id)
		return execErr
	})
}
