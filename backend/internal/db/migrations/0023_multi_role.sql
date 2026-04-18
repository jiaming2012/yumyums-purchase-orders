-- +goose Up
BEGIN;

-- 1. users.role TEXT -> TEXT[] (must DROP CHECK first)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ALTER COLUMN role TYPE TEXT[] USING ARRAY[role];
ALTER TABLE users RENAME COLUMN role TO roles;
ALTER TABLE users ADD CONSTRAINT users_roles_check CHECK (
  roles <@ ARRAY['admin','manager','team_member']::TEXT[]
  AND array_length(roles, 1) > 0
);

-- 2. ob_templates.role TEXT -> TEXT[] (nullable, no CHECK to drop)
ALTER TABLE ob_templates ALTER COLUMN role TYPE TEXT[] USING CASE WHEN role IS NULL THEN NULL ELSE ARRAY[role] END;
ALTER TABLE ob_templates RENAME COLUMN role TO roles;

COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_templates RENAME COLUMN roles TO role;
ALTER TABLE ob_templates ALTER COLUMN role TYPE TEXT USING role[1];

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_roles_check;
ALTER TABLE users RENAME COLUMN roles TO role;
ALTER TABLE users ALTER COLUMN role TYPE TEXT USING role[1];
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin','manager','team_member'));
COMMIT;
