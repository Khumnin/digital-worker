// internal/repository/postgres/audit_repo.go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		args := []interface{}{}
		conds := []string{"a.archived = false"}

		if filter.EventType != nil && *filter.EventType != "" {
			args = append(args, *filter.EventType)
			conds = append(conds, fmt.Sprintf("a.event_type = $%d", len(args)))
		}
		if filter.ActorID != nil {
			args = append(args, *filter.ActorID)
			conds = append(conds, fmt.Sprintf("a.actor_id = $%d", len(args)))
		}
		if filter.TargetUserID != nil {
			args = append(args, *filter.TargetUserID)
			conds = append(conds, fmt.Sprintf("a.target_user_id = $%d", len(args)))
		}
		if filter.From != nil {
			args = append(args, *filter.From)
			conds = append(conds, fmt.Sprintf("a.occurred_at >= $%d", len(args)))
		}
		if filter.To != nil {
			args = append(args, *filter.To)
			conds = append(conds, fmt.Sprintf("a.occurred_at <= $%d", len(args)))
		}

		where := "WHERE " + strings.Join(conds, " AND ")

		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM audit_log a %s", where)
		if countErr := conn.QueryRow(ctx, countSQL, args...).Scan(&total); countErr != nil {
			return fmt.Errorf("count audit log: %w", countErr)
		}

		limit := filter.Limit
		if limit <= 0 {
			limit = 50
		}

		args = append(args, limit, filter.Offset)
		dataSQL := fmt.Sprintf(`
			SELECT a.id, a.event_type, a.actor_id, a.actor_ip, a.actor_ua,
			       a.target_user_id, a.metadata, a.occurred_at, a.archived,
			       actor_u.email AS actor_email,
			       target_u.email AS target_email
			FROM audit_log a
			LEFT JOIN users actor_u  ON actor_u.id  = a.actor_id AND actor_u.deleted_at IS NULL
			LEFT JOIN users target_u ON target_u.id = a.target_user_id AND target_u.deleted_at IS NULL
			%s
			ORDER BY a.occurred_at DESC
			LIMIT $%d OFFSET $%d
		`, where, len(args)-1, len(args))

		rows, queryErr := conn.Query(ctx, dataSQL, args...)
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
				&e.ActorEmail, &e.TargetEmail,
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

// AnonymizeActor replaces all actor_id references matching userID with tombstoneID.
// This is called as part of GDPR right-to-erasure processing.
func (r *PostgresAuditRepo) AnonymizeActor(ctx context.Context, userID uuid.UUID, tombstoneID uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}
	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx,
			`UPDATE audit_log SET actor_id = $2 WHERE actor_id = $1`,
			userID, tombstoneID,
		)
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
