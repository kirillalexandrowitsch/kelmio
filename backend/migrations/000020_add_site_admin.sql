-- +goose Up
-- Global (site) administrator role for managing organizations (PLAT-006).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_site_admin boolean NOT NULL DEFAULT false;

-- Existing workspace admins become site administrators so an upgraded
-- installation keeps a usable global administrator.
UPDATE users
SET is_site_admin = true
WHERE id IN (
  SELECT user_id FROM workspace_members WHERE role = 'admin'
);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS is_site_admin;
