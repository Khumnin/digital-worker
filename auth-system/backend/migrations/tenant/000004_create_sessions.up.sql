-- migrations/tenant/000004_create_sessions.up.sql
-- Per-tenant refresh token sessions (ADR-004).
-- Token reuse detection via family_id: revoking one token in a family revokes all.

CREATE TABLE IF NOT EXISTS sessions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT        NOT NULL UNIQUE,
    family_id           UUID        NOT NULL,
    ip_address          TEXT        NOT NULL DEFAULT '',
    user_agent          TEXT        NOT NULL DEFAULT '',
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    is_revoked          BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX idx_sessions_user_id            ON sessions (user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON sessions (refresh_token_hash);
CREATE INDEX idx_sessions_family_id          ON sessions (family_id);
CREATE INDEX idx_sessions_expires_at         ON sessions (expires_at) WHERE is_revoked = false;
