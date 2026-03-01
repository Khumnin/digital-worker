-- migrations/tenant/000006_create_email_verification_tokens.up.sql
-- Stores SHA-256 hashed email verification tokens (raw token sent only in email).

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN     NOT NULL DEFAULT false,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_email_verification_tokens_user_id    ON email_verification_tokens (user_id);
CREATE INDEX idx_email_verification_tokens_token_hash ON email_verification_tokens (token_hash);
CREATE INDEX idx_email_verification_tokens_expires_at ON email_verification_tokens (expires_at) WHERE used = false;
