-- migrations/tenant/000017_add_module_to_roles.up.sql
-- Adds module scoping and system-role flag to the roles table.
-- module: NULL for system/tenant-level roles; set to module name (e.g. 'recruit') for module roles.
-- is_system: TRUE for seeded immutable roles; prevents deletion via the RBAC admin API.

ALTER TABLE roles ADD COLUMN IF NOT EXISTS module VARCHAR(100) NULL;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- Mark existing seeded system roles as is_system=true.
UPDATE roles SET is_system = true WHERE name IN ('user', 'admin', 'super_admin', 'applicant');
