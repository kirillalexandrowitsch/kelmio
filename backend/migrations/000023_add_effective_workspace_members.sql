-- +goose Up
-- Effective workspace membership (PLAT-007). A user's effective access to a
-- workspace is the maximum of their direct membership and any workspace role
-- assignments granted to them directly or through a group. This view is the
-- single source of truth for resolving who belongs to a workspace and with
-- which role. When no role assignments exist it is identical to
-- workspace_members, so existing single-workspace installs are unaffected.
-- 'admin' outranks 'member'. CREATE OR REPLACE keeps the migration idempotent.

CREATE OR REPLACE VIEW effective_workspace_members AS
WITH sourced AS (
  SELECT workspace_id, user_id, role, joined_at
  FROM workspace_members

  UNION ALL

  SELECT ra.scope_id AS workspace_id, ra.subject_id AS user_id, ra.role, ra.created_at AS joined_at
  FROM role_assignments ra
  WHERE ra.scope = 'workspace' AND ra.subject_type = 'user'

  UNION ALL

  SELECT ra.scope_id AS workspace_id, gm.user_id, ra.role, ra.created_at AS joined_at
  FROM role_assignments ra
  JOIN group_members gm ON gm.group_id = ra.subject_id
  WHERE ra.scope = 'workspace' AND ra.subject_type = 'group'
)
SELECT
  workspace_id,
  user_id,
  CASE WHEN bool_or(role = 'admin') THEN 'admin' ELSE 'member' END AS role,
  min(joined_at) AS joined_at
FROM sourced
GROUP BY workspace_id, user_id;

-- +goose Down
DROP VIEW IF EXISTS effective_workspace_members;
