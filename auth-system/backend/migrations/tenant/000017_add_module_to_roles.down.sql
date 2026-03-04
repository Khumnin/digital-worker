-- migrations/tenant/000017_add_module_to_roles.down.sql
ALTER TABLE roles DROP COLUMN IF EXISTS module;
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
