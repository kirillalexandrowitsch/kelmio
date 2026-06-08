-- +goose Up
CREATE TABLE IF NOT EXISTS project_workflow_statuses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  key text NOT NULL,
  name text NOT NULL,
  color text NOT NULL,
  category text NOT NULL,
  position integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  archived_at timestamptz,
  CONSTRAINT project_workflow_statuses_id_project_unique UNIQUE (id, project_id),
  CONSTRAINT project_workflow_statuses_project_key_unique UNIQUE (project_id, key),
  CONSTRAINT project_workflow_statuses_key_valid CHECK (
    key ~ '^[a-z][a-z0-9_]{0,31}$'
  ),
  CONSTRAINT project_workflow_statuses_name_valid CHECK (
    btrim(name) <> ''
    AND char_length(name) <= 60
  ),
  CONSTRAINT project_workflow_statuses_color_valid CHECK (
    color ~ '^#[0-9A-Fa-f]{6}$'
  ),
  CONSTRAINT project_workflow_statuses_category_valid CHECK (
    category IN ('backlog', 'todo', 'in_progress', 'done')
  ),
  CONSTRAINT project_workflow_statuses_position_valid CHECK (position >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_workflow_statuses_active_name
  ON project_workflow_statuses(project_id, lower(name))
  WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_project_workflow_statuses_project_position
  ON project_workflow_statuses(project_id, position, id);

CREATE INDEX IF NOT EXISTS idx_project_workflow_statuses_project_category
  ON project_workflow_statuses(project_id, category)
  WHERE archived_at IS NULL;

CREATE TABLE IF NOT EXISTS project_workflow_transitions (
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  from_status_id uuid NOT NULL,
  to_status_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, from_status_id, to_status_id),
  CONSTRAINT project_workflow_transitions_from_status_fk
    FOREIGN KEY (from_status_id, project_id)
    REFERENCES project_workflow_statuses(id, project_id)
    ON DELETE CASCADE,
  CONSTRAINT project_workflow_transitions_to_status_fk
    FOREIGN KEY (to_status_id, project_id)
    REFERENCES project_workflow_statuses(id, project_id)
    ON DELETE CASCADE,
  CONSTRAINT project_workflow_transitions_not_self CHECK (
    from_status_id <> to_status_id
  )
);

CREATE INDEX IF NOT EXISTS idx_project_workflow_transitions_from_status
  ON project_workflow_transitions(from_status_id, to_status_id);

CREATE INDEX IF NOT EXISTS idx_project_workflow_transitions_to_status
  ON project_workflow_transitions(to_status_id, from_status_id);

CREATE OR REPLACE FUNCTION initialize_default_project_workflow(target_project_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO project_workflow_statuses (
    project_id,
    key,
    name,
    color,
    category,
    position
  )
  VALUES
    (target_project_id, 'backlog', 'Backlog', '#64748b', 'backlog', 100),
    (target_project_id, 'todo', 'Todo', '#3b82f6', 'todo', 200),
    (target_project_id, 'in_progress', 'In progress', '#f59e0b', 'in_progress', 300),
    (target_project_id, 'blocked', 'Blocked', '#dc2626', 'in_progress', 400),
    (target_project_id, 'done', 'Done', '#16a34a', 'done', 500)
  ON CONFLICT (project_id, key) DO NOTHING;

  INSERT INTO project_workflow_transitions (
    project_id,
    from_status_id,
    to_status_id
  )
  SELECT
    target_project_id,
    source_status.id,
    target_status.id
  FROM project_workflow_statuses source_status
  CROSS JOIN project_workflow_statuses target_status
  WHERE source_status.project_id = target_project_id
    AND target_status.project_id = target_project_id
    AND source_status.archived_at IS NULL
    AND target_status.archived_at IS NULL
    AND source_status.id <> target_status.id
  ON CONFLICT (project_id, from_status_id, to_status_id) DO NOTHING;
END;
$$;

CREATE OR REPLACE FUNCTION initialize_default_project_workflow_after_project_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM initialize_default_project_workflow(NEW.id);
  RETURN NEW;
END;
$$;

CREATE TRIGGER projects_initialize_default_workflow
AFTER INSERT ON projects
FOR EACH ROW
EXECUTE FUNCTION initialize_default_project_workflow_after_project_insert();

SELECT initialize_default_project_workflow(id)
FROM projects;

CREATE OR REPLACE FUNCTION protect_project_workflow_status_identity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.key <> OLD.key OR NEW.project_id <> OLD.project_id THEN
    RAISE EXCEPTION 'workflow status key and project_id are immutable'
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER project_workflow_statuses_protect_identity
BEFORE UPDATE OF key, project_id ON project_workflow_statuses
FOR EACH ROW
EXECUTE FUNCTION protect_project_workflow_status_identity();

ALTER TABLE issues
  ADD COLUMN IF NOT EXISTS workflow_status_id uuid;

UPDATE issues issue
SET workflow_status_id = workflow_status.id
FROM project_workflow_statuses workflow_status
WHERE workflow_status.project_id = issue.project_id
  AND workflow_status.key = issue.status
  AND workflow_status.archived_at IS NULL
  AND issue.workflow_status_id IS NULL;

ALTER TABLE issues
  ALTER COLUMN workflow_status_id SET NOT NULL;

ALTER TABLE issues
  ADD CONSTRAINT issues_workflow_status_project_fk
  FOREIGN KEY (workflow_status_id, project_id)
  REFERENCES project_workflow_statuses(id, project_id);

CREATE INDEX IF NOT EXISTS idx_issues_workflow_status_id
  ON issues(workflow_status_id);

CREATE INDEX IF NOT EXISTS idx_issues_project_workflow_status_created_id
  ON issues(project_id, workflow_status_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_issues_sprint_workflow_status_created_id
  ON issues(sprint_id, workflow_status_id, created_at DESC, id DESC);

CREATE OR REPLACE FUNCTION sync_issue_legacy_workflow_status()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  SELECT workflow_status.id
  INTO NEW.workflow_status_id
  FROM project_workflow_statuses workflow_status
  WHERE workflow_status.project_id = NEW.project_id
    AND workflow_status.key = NEW.status
    AND workflow_status.archived_at IS NULL;

  IF NEW.workflow_status_id IS NULL THEN
    RAISE EXCEPTION
      'active workflow status with key "%" was not found for project %',
      NEW.status,
      NEW.project_id
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER issues_sync_legacy_workflow_status
BEFORE INSERT OR UPDATE ON issues
FOR EACH ROW
EXECUTE FUNCTION sync_issue_legacy_workflow_status();

-- +goose Down
DROP TRIGGER IF EXISTS issues_sync_legacy_workflow_status ON issues;
DROP FUNCTION IF EXISTS sync_issue_legacy_workflow_status();

DROP INDEX IF EXISTS idx_issues_sprint_workflow_status_created_id;
DROP INDEX IF EXISTS idx_issues_project_workflow_status_created_id;
DROP INDEX IF EXISTS idx_issues_workflow_status_id;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_workflow_status_project_fk;

ALTER TABLE issues
  DROP COLUMN IF EXISTS workflow_status_id;

DROP TRIGGER IF EXISTS project_workflow_statuses_protect_identity
  ON project_workflow_statuses;
DROP FUNCTION IF EXISTS protect_project_workflow_status_identity();

DROP TRIGGER IF EXISTS projects_initialize_default_workflow ON projects;
DROP FUNCTION IF EXISTS initialize_default_project_workflow_after_project_insert();
DROP FUNCTION IF EXISTS initialize_default_project_workflow(uuid);

DROP INDEX IF EXISTS idx_project_workflow_transitions_to_status;
DROP INDEX IF EXISTS idx_project_workflow_transitions_from_status;
DROP TABLE IF EXISTS project_workflow_transitions;

DROP INDEX IF EXISTS idx_project_workflow_statuses_project_category;
DROP INDEX IF EXISTS idx_project_workflow_statuses_project_position;
DROP INDEX IF EXISTS idx_project_workflow_statuses_active_name;
DROP TABLE IF EXISTS project_workflow_statuses;
