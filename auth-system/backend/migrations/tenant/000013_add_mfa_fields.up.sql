-- migrations/tenant/000013_add_mfa_fields.up.sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_totp_secret TEXT;
