-- +goose Up
ALTER TABLE issues
  ADD COLUMN IF NOT EXISTS parent_issue_id uuid REFERENCES issues(id) ON DELETE CASCADE;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_issue_type_check;

ALTER TABLE issues
  ADD CONSTRAINT issues_issue_type_check
  CHECK (issue_type IN ('task', 'bug', 'story', 'epic', 'subtask'));

ALTER TABLE issues
  ADD CONSTRAINT issues_subtask_parent_required
  CHECK (issue_type <> 'subtask' OR parent_issue_id IS NOT NULL);

ALTER TABLE issues
  ADD CONSTRAINT issues_epic_parent_forbidden
  CHECK (issue_type <> 'epic' OR parent_issue_id IS NULL);

ALTER TABLE issues
  ADD CONSTRAINT issues_parent_not_self
  CHECK (parent_issue_id IS NULL OR parent_issue_id <> id);

CREATE INDEX IF NOT EXISTS idx_issues_parent_issue_id ON issues(parent_issue_id);

-- +goose Down
ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_subtask_parent_required;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_epic_parent_forbidden;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_parent_not_self;

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_issue_type_check;

DROP INDEX IF EXISTS idx_issues_parent_issue_id;

ALTER TABLE issues
  DROP COLUMN IF EXISTS parent_issue_id;

ALTER TABLE issues
  ADD CONSTRAINT issues_issue_type_check
  CHECK (issue_type IN ('task', 'bug', 'story'));
