-- migrations/tenant/000018_seed_module_roles.up.sql
-- Seeds recruitment module roles into every tenant schema.
-- These roles are created as is_system=true so they cannot be deleted via the RBAC API.
-- Recruitment backend assigns them to users via the RBAC API at client onboarding time.

INSERT INTO roles (id, name, description, module, is_system, created_at)
VALUES
    (gen_random_uuid(), 'recruiter',       'Can manage job postings and candidates',  'recruit', true, now()),
    (gen_random_uuid(), 'hiring_manager',  'Can approve/reject candidates',           'recruit', true, now()),
    (gen_random_uuid(), 'interviewer',     'Can conduct and record interviews',        'recruit', true, now())
ON CONFLICT (name) DO NOTHING;
