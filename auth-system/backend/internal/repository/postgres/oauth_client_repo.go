// internal/repository/postgres/oauth_client_repo.go
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

type postgresOAuthClientRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresOAuthClientRepo(pool *pgxpool.Pool) domain.OAuthClientRepository {
	return &postgresOAuthClientRepo{pool: pool}
}

func (r *postgresOAuthClientRepo) Create(ctx context.Context, c *domain.OAuthClient) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO oauth_clients
				(id, client_id, client_secret, name, redirect_uris, scopes, grant_types, is_confidential, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now())`,
			c.ID, c.ClientID, c.ClientSecret, c.Name,
			c.RedirectURIs, c.Scopes, c.GrantTypes, c.IsConfidential, c.CreatedBy,
		)
		return err
	})
}

func (r *postgresOAuthClientRepo) FindByClientID(ctx context.Context, clientID string) (*domain.OAuthClient, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var client domain.OAuthClient
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, client_id, client_secret, name,
			       redirect_uris, scopes, grant_types, is_confidential,
			       created_by, created_at, updated_at
			FROM oauth_clients
			WHERE client_id = $1`, clientID)
		return row.Scan(
			&client.ID, &client.ClientID, &client.ClientSecret, &client.Name,
			&client.RedirectURIs, &client.Scopes, &client.GrantTypes, &client.IsConfidential,
			&client.CreatedBy, &client.CreatedAt, &client.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuthClientNotFound
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *postgresOAuthClientRepo) ListByCreator(ctx context.Context, createdBy uuid.UUID) ([]*domain.OAuthClient, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var clients []*domain.OAuthClient
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, client_id, client_secret, name,
			       redirect_uris, scopes, grant_types, is_confidential,
			       created_by, created_at, updated_at
			FROM oauth_clients
			WHERE created_by = $1
			ORDER BY created_at DESC`, createdBy)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c domain.OAuthClient
			if err := rows.Scan(
				&c.ID, &c.ClientID, &c.ClientSecret, &c.Name,
				&c.RedirectURIs, &c.Scopes, &c.GrantTypes, &c.IsConfidential,
				&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
			); err != nil {
				return err
			}
			clients = append(clients, &c)
		}
		return rows.Err()
	})
	return clients, err
}
