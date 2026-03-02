-- migrations/global/000001_create_tenants.up.sql
-- Global tenants table: one row per tenant.
-- Schema-per-tenant model (ADR-001): each tenant gets an isolated PostgreSQL schema.

CREATE TABLE IF NOT EXISTS tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    schema_name TEXT        NOT NULL UNIQUE,
    admin_email TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'suspended', 'deleted')),
    config      JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug        ON tenants (slug)        WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_schema_name ON tenants (schema_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status      ON tenants (status)      WHERE deleted_at IS NULL;
