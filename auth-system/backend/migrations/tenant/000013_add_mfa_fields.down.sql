-- migrations/tenant/000013_add_mfa_fields.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS mfa_totp_secret;
