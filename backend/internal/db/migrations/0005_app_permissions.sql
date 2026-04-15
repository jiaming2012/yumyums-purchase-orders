-- +goose Up
BEGIN;
CREATE TABLE app_permissions (
  id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_id  UUID NOT NULL REFERENCES hq_apps(id) ON DELETE CASCADE,
  role    TEXT CHECK (role IN ('admin', 'manager', 'team_member')),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT role_or_user CHECK (
    (role IS NOT NULL AND user_id IS NULL) OR
    (role IS NULL AND user_id IS NOT NULL)
  )
);
CREATE UNIQUE INDEX app_permissions_role_idx ON app_permissions(app_id, role) WHERE role IS NOT NULL;
CREATE UNIQUE INDEX app_permissions_user_idx ON app_permissions(app_id, user_id) WHERE user_id IS NOT NULL;
COMMIT;

-- +goose Down
DROP TABLE app_permissions;
