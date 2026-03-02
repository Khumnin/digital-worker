// internal/repository/postgres/credential_repo.go
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
)

// PostgresCredentialRepo implements domain.TenantCredentialRepository.
type PostgresCredentialRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresCredentialRepo(pool *pgxpool.Pool) *PostgresCredentialRepo {
	return &PostgresCredentialRepo{pool: pool}
}

func (r *PostgresCredentialRepo) Create(ctx context.Context, tenantID uuid.UUID, clientID, secretHash string) (*domain.TenantAPICredential, error) {
	cred := &domain.TenantAPICredential{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tenant_api_credentials (tenant_id, client_id, client_secret_hash)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, client_id, client_secret_hash, created_at, rotated_at, revoked_at
	`, tenantID, clientID, secretHash).Scan(
		&cred.ID, &cred.TenantID, &cred.ClientID, &cred.ClientSecretHash,
		&cred.CreatedAt, &cred.RotatedAt, &cred.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tenant api credential: %w", err)
	}
	return cred, nil
}

func (r *PostgresCredentialRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantAPICredential, error) {
	cred := &domain.TenantAPICredential{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, client_id, client_secret_hash, created_at, rotated_at, revoked_at
		FROM tenant_api_credentials
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID).Scan(
		&cred.ID, &cred.TenantID, &cred.ClientID, &cred.ClientSecretHash,
		&cred.CreatedAt, &cred.RotatedAt, &cred.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCredentialNotFound
		}
		return nil, fmt.Errorf("find tenant api credential by tenant: %w", err)
	}
	return cred, nil
}

func (r *PostgresCredentialRepo) FindByClientID(ctx context.Context, clientID string) (*domain.TenantAPICredential, error) {
	cred := &domain.TenantAPICredential{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, client_id, client_secret_hash, created_at, rotated_at, revoked_at
		FROM tenant_api_credentials
		WHERE client_id = $1
	`, clientID).Scan(
		&cred.ID, &cred.TenantID, &cred.ClientID, &cred.ClientSecretHash,
		&cred.CreatedAt, &cred.RotatedAt, &cred.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCredentialNotFound
		}
		return nil, fmt.Errorf("find tenant api credential by client_id: %w", err)
	}
	return cred, nil
}

func (r *PostgresCredentialRepo) Rotate(ctx context.Context, tenantID uuid.UUID, newClientID, newSecretHash string) (*domain.TenantAPICredential, error) {
	now := time.Now()

	// Revoke all existing credentials for this tenant, then insert the new one
	// in a single transaction.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		UPDATE tenant_api_credentials SET revoked_at = $1
		WHERE tenant_id = $2 AND revoked_at IS NULL
	`, now, tenantID); err != nil {
		return nil, fmt.Errorf("revoke existing credentials: %w", err)
	}

	cred := &domain.TenantAPICredential{}
	if err := tx.QueryRow(ctx, `
		INSERT INTO tenant_api_credentials (tenant_id, client_id, client_secret_hash, rotated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, client_id, client_secret_hash, created_at, rotated_at, revoked_at
	`, tenantID, newClientID, newSecretHash, now).Scan(
		&cred.ID, &cred.TenantID, &cred.ClientID, &cred.ClientSecretHash,
		&cred.CreatedAt, &cred.RotatedAt, &cred.RevokedAt,
	); err != nil {
		return nil, fmt.Errorf("insert rotated credential: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credential rotation: %w", err)
	}

	return cred, nil
}
