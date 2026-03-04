-- migrations/tenant/000016_add_applicant_role.up.sql
-- Adds the applicant system role for cross-tenant identity support.
-- Applicants authenticate against the platform tenant and their JWT is accepted
-- by downstream modules (e.g. Recruitment) via JWKS verification.
-- is_system = true prevents deletion via the RBAC admin API.

INSERT INTO roles (id, name, description, is_system, created_at)
VALUES (
    gen_random_uuid(),
    'applicant',
    'External applicant — centralized identity for job seekers across all Tigersoft clients.',
    true,
    now()
)
ON CONFLICT (name) DO NOTHING;
