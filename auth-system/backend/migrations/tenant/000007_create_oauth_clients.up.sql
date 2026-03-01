-- migrations/tenant/000007_create_oauth_clients.up.sql
-- OAuth 2.0 client registrations per tenant. Sprint 5 placeholder.

CREATE TABLE IF NOT EXISTS oauth_clients (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id       TEXT        NOT NULL UNIQUE,
    client_secret   TEXT        NOT NULL,
    name            TEXT        NOT NULL,
    redirect_uris   JSONB       NOT NULL DEFAULT '[]',
    scopes          JSONB       NOT NULL DEFAULT '[]',
    grant_types     JSONB       NOT NULL DEFAULT '["authorization_code"]',
    is_confidential BOOLEAN     NOT NULL DEFAULT true,
    created_by      UUID        REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_clients_client_id ON oauth_clients (client_id);
