-- migrations/tenant/000011_create_tenant_config.up.sql
-- Per-tenant runtime configuration key-value store.
-- Used for fine-grained config that doesn't belong in the global tenants.config JSONB.

CREATE TABLE IF NOT EXISTS tenant_config (
    key         TEXT        PRIMARY KEY,
    value       JSONB       NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
