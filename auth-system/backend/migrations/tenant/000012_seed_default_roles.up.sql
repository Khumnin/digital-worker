-- migrations/tenant/000012_seed_default_roles.up.sql
-- Seeds system roles into every new tenant schema.
-- is_system = true prevents deletion via the RBAC admin API.

INSERT INTO roles (id, name, description, is_system, created_at)
VALUES
    (gen_random_uuid(), 'user',        'Standard user — can authenticate and manage their own profile.', true, now()),
    (gen_random_uuid(), 'admin',       'Tenant administrator — can manage users and roles within the tenant.', true, now()),
    (gen_random_uuid(), 'super_admin', 'Super administrator — global platform access. Assigned by platform ops only.', true, now())
ON CONFLICT (name) DO NOTHING;
