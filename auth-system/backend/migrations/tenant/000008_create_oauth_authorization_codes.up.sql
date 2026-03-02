-- migrations/tenant/000008_create_oauth_authorization_codes.up.sql
-- OAuth 2.0 authorization codes (PKCE). Sprint 5 placeholder.

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash           TEXT        NOT NULL UNIQUE,
    client_id           TEXT        NOT NULL,
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri        TEXT        NOT NULL,
    scopes              JSONB       NOT NULL DEFAULT '[]',
    code_challenge      TEXT,
    code_challenge_method TEXT,
    expires_at          TIMESTAMPTZ NOT NULL,
    used                BOOLEAN     NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_codes_code_hash  ON oauth_authorization_codes (code_hash);
CREATE INDEX idx_oauth_codes_expires_at ON oauth_authorization_codes (expires_at) WHERE used = false;
