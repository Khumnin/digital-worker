-- docker/postgres/init.sql
-- Run once when the postgres container is first created.
-- Creates extensions required by the application.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";      -- Case-insensitive text (optional)

-- Create the global schema (public is default and already exists).
-- The tenant schemas are created dynamically by the TenantProvisioner.
