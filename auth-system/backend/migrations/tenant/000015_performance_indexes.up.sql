-- migrations/tenant/000015_performance_indexes.up.sql
-- PERF-04: Indexes for hot query paths identified during Sprint 8 load analysis.

-- Login: email lookup is the most frequent read path; partial index excludes
-- soft-deleted rows so the planner uses a smaller, more selective index.
CREATE INDEX IF NOT EXISTS idx_users_email
    ON users (email)
    WHERE deleted_at IS NULL;

-- Session refresh: token hash lookup on the hot refresh path; partial index
-- on active sessions only keeps the index compact.
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token_hash
    ON sessions (refresh_token_hash)
    WHERE revoked_at IS NULL;

-- Audit log: actor-scoped queries (admin views of a specific user's actions)
-- and time-range filtering both benefit from this compound index.
CREATE INDEX IF NOT EXISTS idx_audit_log_actor_id
    ON audit_log (actor_id, occurred_at DESC);

-- Audit log: event_type filtering for compliance reports and alert dashboards.
CREATE INDEX IF NOT EXISTS idx_audit_log_event_type
    ON audit_log (event_type, occurred_at DESC);

-- OAuth authorization codes: user_id lookup for GDPR erasure and password-change
-- invalidation; previously required a full table scan.
CREATE INDEX IF NOT EXISTS idx_oauth_codes_user_id
    ON oauth_authorization_codes (user_id);

-- Notes:
--   idx_mfa_backup_codes_user_id  — already created in migration 000014
--   idx_oauth_social_accounts_user_id — already created in migration 000009
--   Redis handles rate limiter key lookups entirely in-memory.
