-- migrations/tenant/000009_create_oauth_social_accounts.up.sql
-- Links social provider accounts to local users. Sprint 6 placeholder.

CREATE TABLE IF NOT EXISTS oauth_social_accounts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT        NOT NULL,
    provider_user_id TEXT       NOT NULL,
    email           TEXT,
    access_token    TEXT,
    refresh_token   TEXT,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT oauth_social_accounts_unique UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_oauth_social_accounts_user_id ON oauth_social_accounts (user_id);
