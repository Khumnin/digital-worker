-- migrations/global/000002_create_global_audit_log.up.sql
-- Global audit log: captures super-admin actions on the global schema.
-- Per-tenant audit events are stored in each tenant's own audit_log table.

CREATE TABLE IF NOT EXISTS global_audit_log (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    TEXT        NOT NULL,
    actor_id      UUID,
    actor_ip      TEXT        NOT NULL DEFAULT '',
    actor_ua      TEXT        NOT NULL DEFAULT '',
    target_id     UUID,
    metadata      JSONB       NOT NULL DEFAULT '{}',
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived      BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX idx_global_audit_log_event_type  ON global_audit_log (event_type);
CREATE INDEX idx_global_audit_log_actor_id    ON global_audit_log (actor_id)  WHERE actor_id IS NOT NULL;
CREATE INDEX idx_global_audit_log_occurred_at ON global_audit_log (occurred_at DESC);
