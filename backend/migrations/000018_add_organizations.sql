-- +goose Up
-- Organization and multi-workspace identity foundation (PLAT-006 / PLAT-007).
-- Additive and safe to run against an existing single-workspace installation:
-- new columns are nullable here and backfilled below; NOT NULL enforcement is
-- deferred until every write path sets them.

CREATE TABLE IF NOT EXISTS organizations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  slug text NOT NULL UNIQUE,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_members (
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('org_admin', 'org_member')),
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id, user_id)
);

ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived'));
ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS slug text;

-- Ensure a default organization for existing single-workspace installations.
INSERT INTO organizations (name, slug)
SELECT 'Default Organization', 'default'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'default');

-- Attach existing workspaces to the default organization.
UPDATE workspaces
SET organization_id = (SELECT id FROM organizations WHERE slug = 'default')
WHERE organization_id IS NULL;

-- Backfill unique, readable workspace slugs within the organization.
WITH numbered AS (
  SELECT
    id,
    coalesce(
      nullif(trim(both '-' from regexp_replace(lower(trim(name)), '[^a-z0-9]+', '-', 'g')), ''),
      'workspace'
    ) AS base,
    row_number() OVER (
      PARTITION BY organization_id,
        coalesce(
          nullif(trim(both '-' from regexp_replace(lower(trim(name)), '[^a-z0-9]+', '-', 'g')), ''),
          'workspace'
        )
      ORDER BY created_at, id
    ) AS rn
  FROM workspaces
  WHERE slug IS NULL
)
UPDATE workspaces w
SET slug = CASE WHEN n.rn = 1 THEN n.base ELSE n.base || '-' || n.rn END
FROM numbered n
WHERE w.id = n.id;

-- Backfill organization memberships from existing workspace memberships:
-- a workspace admin becomes an organization admin, everyone else an org member.
INSERT INTO organization_members (organization_id, user_id, role)
SELECT
  w.organization_id,
  wm.user_id,
  CASE WHEN bool_or(wm.role = 'admin') THEN 'org_admin' ELSE 'org_member' END
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE w.organization_id IS NOT NULL
GROUP BY w.organization_id, wm.user_id
ON CONFLICT (organization_id, user_id) DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_organization_slug
  ON workspaces(organization_id, slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_organization_id
  ON workspaces(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_user_id
  ON organization_members(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_organization_members_user_id;
DROP INDEX IF EXISTS idx_workspaces_organization_id;
DROP INDEX IF EXISTS idx_workspaces_organization_slug;
ALTER TABLE workspaces DROP COLUMN IF EXISTS slug;
ALTER TABLE workspaces DROP COLUMN IF EXISTS status;
ALTER TABLE workspaces DROP COLUMN IF EXISTS organization_id;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
