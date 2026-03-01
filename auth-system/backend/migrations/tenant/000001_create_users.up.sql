-- migrations/tenant/000001_create_users.up.sql
-- Per-tenant users table. Each tenant schema gets its own isolated copy.
-- schema-per-tenant isolation (ADR-001, ADR-002).

CREATE TABLE IF NOT EXISTS users (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email               TEXT        NOT NULL,
    password_hash       TEXT        NOT NULL DEFAULT '',
    status              TEXT        NOT NULL DEFAULT 'unverified'
                                    CHECK (status IN ('unverified', 'active', 'disabled', 'deleted')),
    first_name          TEXT        NOT NULL DEFAULT '',
    last_name           TEXT        NOT NULL DEFAULT '',
    mfa_enabled         BOOLEAN     NOT NULL DEFAULT false,
    mfa_totp_secret     TEXT,
    failed_login_count  INT         NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    email_verified_at   TIMESTAMPTZ,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,

    CONSTRAINT users_email_unique UNIQUE (email)
);

-- Partial index: only non-deleted users participate in lookups.
CREATE INDEX idx_users_email  ON users (email)  WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users (status) WHERE deleted_at IS NULL;
