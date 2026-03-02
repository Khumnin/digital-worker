-- migrations/tenant/000010_create_audit_log.up.sql
-- Per-tenant immutable audit log. Append-only by application convention.
-- Retention: 1 year active + cold archive (ADR-007).

CREATE TABLE IF NOT EXISTS audit_log (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      TEXT        NOT NULL,
    actor_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
    actor_ip        TEXT        NOT NULL DEFAULT '',
    actor_ua        TEXT        NOT NULL DEFAULT '',
    target_user_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived        BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX idx_audit_log_event_type    ON audit_log (event_type);
CREATE INDEX idx_audit_log_actor_id      ON audit_log (actor_id)      WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_log_target_id     ON audit_log (target_user_id) WHERE target_user_id IS NOT NULL;
CREATE INDEX idx_audit_log_occurred_at   ON audit_log (occurred_at DESC);
CREATE INDEX idx_audit_log_not_archived  ON audit_log (occurred_at DESC) WHERE archived = false;
