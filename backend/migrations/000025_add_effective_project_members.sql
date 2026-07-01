-- +goose Up
-- Effective project membership (PLAT-007). A user's effective role on a project
-- is the maximum of their direct project membership and any project role
-- assignments granted directly or through a group. This view is the single
-- source of truth for resolving project roles. When no project role assignments
-- exist it is identical to project_members, so existing installs are unaffected.
-- Project roles rank lead > contributor > viewer. CREATE OR REPLACE keeps the
-- migration idempotent.

CREATE OR REPLACE VIEW effective_project_members AS
WITH sourced AS (
  SELECT project_id, user_id, role
  FROM project_members

  UNION ALL

  SELECT ra.scope_id AS project_id, ra.subject_id AS user_id, ra.role
  FROM role_assignments ra
  WHERE ra.scope = 'project' AND ra.subject_type = 'user'

  UNION ALL

  SELECT ra.scope_id AS project_id, gm.user_id, ra.role
  FROM role_assignments ra
  JOIN group_members gm ON gm.group_id = ra.subject_id
  WHERE ra.scope = 'project' AND ra.subject_type = 'group'
)
SELECT
  project_id,
  user_id,
  (ARRAY['lead', 'contributor', 'viewer'])[min(
    CASE role WHEN 'lead' THEN 1 WHEN 'contributor' THEN 2 WHEN 'viewer' THEN 3 END
  )] AS role
FROM sourced
WHERE role IN ('lead', 'contributor', 'viewer')
GROUP BY project_id, user_id;

-- +goose Down
DROP VIEW IF EXISTS effective_project_members;
