-- migrations/tenant/000003_create_user_roles.up.sql
-- Join table: many-to-many between users and roles.

CREATE TABLE IF NOT EXISTS user_roles (
    user_id     UUID        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    role_id     UUID        NOT NULL REFERENCES roles(id)  ON DELETE CASCADE,
    assigned_by UUID,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);
