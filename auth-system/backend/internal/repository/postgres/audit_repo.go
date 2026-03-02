// internal/repository/postgres/audit_repo.go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/domain"
	pgdb "tigersoft/auth-system/internal/infrastructure/postgres"
)

// PostgresAuditRepo implements domain.AuditRepository.
type PostgresAuditRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditRepo(pool *pgxpool.Pool) *PostgresAuditRepo {
	return &PostgresAuditRepo{pool: pool}
}

func (r *PostgresAuditRepo) Append(ctx context.Context, event *domain.AuditEvent) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			INSERT INTO audit_log (id, event_type, actor_id, actor_ip, actor_ua,
			                       target_user_id, metadata, occurred_at, archived)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false)
		`,
			event.ID, string(event.EventType), event.ActorID, event.ActorIP,
			event.ActorUA, event.TargetUserID, metadata, event.OccurredAt,
		)
		return execErr
	})
}

func (r *PostgresAuditRepo) List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditEvent, int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	var events []*domain.AuditEvent
	var total int

	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		if countErr := conn.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log").Scan(&total); countErr != nil {
			return fmt.Errorf("count audit log: %w", countErr)
		}

		limit := filter.Limit
		if limit <= 0 {
			limit = 50
		}

		rows, queryErr := conn.Query(ctx, `
			SELECT id, event_type, actor_id, actor_ip, actor_ua,
			       target_user_id, metadata, occurred_at, archived
			FROM audit_log
			ORDER BY occurred_at DESC
			LIMIT $1 OFFSET $2
		`, limit, filter.Offset)
		if queryErr != nil {
			return fmt.Errorf("query audit log: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.AuditEvent{}
			var metadataJSON []byte
			scanErr := rows.Scan(
				&e.ID, &e.EventType, &e.ActorID, &e.ActorIP, &e.ActorUA,
				&e.TargetUserID, &metadataJSON, &e.OccurredAt, &e.Archived,
			)
			if scanErr != nil {
				return fmt.Errorf("scan audit event: %w", scanErr)
			}
			if len(metadataJSON) > 0 {
				_ = json.Unmarshal(metadataJSON, &e.Metadata)
			}
			events = append(events, e)
		}
		return rows.Err()
	})

	return events, total, err
}

func (r *PostgresAuditRepo) MarkArchived(ctx context.Context, ids []uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE audit_log SET archived = true WHERE id = ANY($1)
		`, ids)
		return execErr
	})
}

func (r *PostgresAuditRepo) ListForArchive(ctx context.Context, cutoff time.Time, limit int) ([]*domain.AuditEvent, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var events []*domain.AuditEvent
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		rows, queryErr := conn.Query(ctx, `
			SELECT id, event_type, actor_id, actor_ip, actor_ua,
			       target_user_id, metadata, occurred_at, archived
			FROM audit_log
			WHERE occurred_at < $1 AND archived = false
			ORDER BY occurred_at ASC
			LIMIT $2
		`, cutoff, limit)
		if queryErr != nil {
			return fmt.Errorf("query archive candidates: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			e := &domain.AuditEvent{}
			var metadataJSON []byte
			scanErr := rows.Scan(
				&e.ID, &e.EventType, &e.ActorID, &e.ActorIP, &e.ActorUA,
				&e.TargetUserID, &metadataJSON, &e.OccurredAt, &e.Archived,
			)
			if scanErr != nil {
				return fmt.Errorf("scan archive candidate: %w", scanErr)
			}
			events = append(events, e)
		}
		return rows.Err()
	})

	return events, err
}
