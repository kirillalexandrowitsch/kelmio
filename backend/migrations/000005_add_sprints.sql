-- +goose Up
ALTER TABLE projects
  ADD CONSTRAINT projects_id_workspace_id_unique UNIQUE (id, workspace_id);

CREATE TABLE IF NOT EXISTS sprints (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL,
  project_id uuid NOT NULL,
  name text NOT NULL,
  goal text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'planned'
    CHECK (status IN ('planned', 'active', 'completed')),
  start_date date,
  end_date date,
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  CONSTRAINT sprints_project_workspace_fk
    FOREIGN KEY (project_id, workspace_id)
    REFERENCES projects(id, workspace_id)
    ON DELETE CASCADE,
  CONSTRAINT sprints_name_not_blank CHECK (btrim(name) <> ''),
  CONSTRAINT sprints_dates_order CHECK (
    start_date IS NULL
    OR end_date IS NULL
    OR start_date <= end_date
  ),
  CONSTRAINT sprints_completed_at_matches_status CHECK (
    (status = 'completed' AND completed_at IS NOT NULL)
    OR (status <> 'completed' AND completed_at IS NULL)
  )
);

ALTER TABLE issues
  ADD COLUMN IF NOT EXISTS sprint_id uuid;

ALTER TABLE issues
  ADD CONSTRAINT issues_sprint_id_fk
  FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sprints_one_active_per_project
  ON sprints(project_id)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_sprints_workspace_id ON sprints(workspace_id);
CREATE INDEX IF NOT EXISTS idx_sprints_project_id ON sprints(project_id);
CREATE INDEX IF NOT EXISTS idx_sprints_status ON sprints(status);
CREATE INDEX IF NOT EXISTS idx_issues_sprint_id ON issues(sprint_id);

-- +goose Down
DROP INDEX IF EXISTS idx_issues_sprint_id;
DROP INDEX IF EXISTS idx_sprints_status;
DROP INDEX IF EXISTS idx_sprints_project_id;
DROP INDEX IF EXISTS idx_sprints_workspace_id;
DROP INDEX IF EXISTS idx_sprints_one_active_per_project;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_sprint_id_fk;

ALTER TABLE issues
  DROP COLUMN IF EXISTS sprint_id;

DROP TABLE IF EXISTS sprints;

ALTER TABLE projects
  DROP CONSTRAINT IF EXISTS projects_id_workspace_id_unique;
