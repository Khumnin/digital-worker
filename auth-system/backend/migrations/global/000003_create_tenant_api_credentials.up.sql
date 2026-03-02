-- Global table for M2M tenant API credentials.
-- client_id is a stable public identifier; client_secret_hash stores a
-- SHA-256 hash of the secret (shown to the user once on creation/rotation).

CREATE TABLE IF NOT EXISTS tenant_api_credentials (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL REFERENCES tenants(id),
    client_id          TEXT        NOT NULL UNIQUE,
    client_secret_hash TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at         TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ
);

CREATE INDEX idx_tenant_api_creds_tenant_id  ON tenant_api_credentials (tenant_id);
CREATE INDEX idx_tenant_api_creds_client_id  ON tenant_api_credentials (client_id);
