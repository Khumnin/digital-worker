-- migrations/tenant/000015_performance_indexes.down.sql
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_sessions_refresh_token_hash;
DROP INDEX IF EXISTS idx_audit_log_actor_id;
DROP INDEX IF EXISTS idx_audit_log_event_type;
DROP INDEX IF EXISTS idx_oauth_codes_user_id;
