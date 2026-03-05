-- migrations/tenant/000018_seed_module_roles.down.sql
DELETE FROM roles WHERE module = 'recruit' AND is_system = true;
