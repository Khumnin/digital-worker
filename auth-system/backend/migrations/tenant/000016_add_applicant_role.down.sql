-- migrations/tenant/000016_add_applicant_role.down.sql
DELETE FROM roles WHERE name = 'applicant' AND is_system = true;
