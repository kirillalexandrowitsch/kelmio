-- +goose Up
CREATE TABLE IF NOT EXISTS project_members (
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, user_id),
  CONSTRAINT project_members_role_valid CHECK (
    role IN ('lead', 'contributor', 'viewer')
  )
);

CREATE INDEX IF NOT EXISTS idx_project_members_user_id
  ON project_members(user_id, project_id);

CREATE INDEX IF NOT EXISTS idx_project_members_project_role
  ON project_members(project_id, role, user_id);

CREATE OR REPLACE FUNCTION validate_project_member_workspace()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM projects project
    JOIN workspace_members workspace_member
      ON workspace_member.workspace_id = project.workspace_id
      AND workspace_member.user_id = NEW.user_id
    JOIN users app_user
      ON app_user.id = workspace_member.user_id
    WHERE project.id = NEW.project_id
      AND app_user.is_active = true
  ) THEN
    RAISE EXCEPTION
      'project member must be an active workspace member'
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER project_members_validate_workspace
BEFORE INSERT OR UPDATE OF project_id, user_id ON project_members
FOR EACH ROW
EXECUTE FUNCTION validate_project_member_workspace();

CREATE OR REPLACE FUNCTION initialize_default_project_members(target_project_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO project_members (
    project_id,
    user_id,
    role
  )
  SELECT
    project.id,
    workspace_member.user_id,
    CASE
      WHEN workspace_member.user_id = project.created_by THEN 'lead'
      ELSE 'contributor'
    END
  FROM projects project
  JOIN workspace_members workspace_member
    ON workspace_member.workspace_id = project.workspace_id
  JOIN users app_user
    ON app_user.id = workspace_member.user_id
  WHERE project.id = target_project_id
    AND app_user.is_active = true
  ON CONFLICT (project_id, user_id) DO UPDATE
  SET role = 'lead',
      updated_at = now()
  WHERE EXCLUDED.role = 'lead'
    AND project_members.role <> 'lead';
END;
$$;

CREATE OR REPLACE FUNCTION initialize_default_project_members_after_project_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM initialize_default_project_members(NEW.id);
  RETURN NEW;
END;
$$;

CREATE TRIGGER projects_initialize_default_members
AFTER INSERT ON projects
FOR EACH ROW
EXECUTE FUNCTION initialize_default_project_members_after_project_insert();

SELECT initialize_default_project_members(id)
FROM projects;

-- +goose Down
DROP TRIGGER IF EXISTS projects_initialize_default_members ON projects;
DROP FUNCTION IF EXISTS initialize_default_project_members_after_project_insert();
DROP FUNCTION IF EXISTS initialize_default_project_members(uuid);

DROP TRIGGER IF EXISTS project_members_validate_workspace ON project_members;
DROP FUNCTION IF EXISTS validate_project_member_workspace();

DROP INDEX IF EXISTS idx_project_members_project_role;
DROP INDEX IF EXISTS idx_project_members_user_id;
DROP TABLE IF EXISTS project_members;
