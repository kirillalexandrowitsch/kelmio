-- +goose Up
-- Reusable organization groups (PLAT-007). Groups belong to an organization and
-- bundle users so access can later be assigned to a group instead of each user
-- individually. Additive and idempotent.

CREATE TABLE IF NOT EXISTS groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS group_members (
  group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  added_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_organization_name
  ON groups(organization_id, name);
CREATE INDEX IF NOT EXISTS idx_groups_organization_id
  ON groups(organization_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user_id
  ON group_members(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_group_members_user_id;
DROP INDEX IF EXISTS idx_groups_organization_id;
DROP INDEX IF EXISTS idx_groups_organization_name;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
