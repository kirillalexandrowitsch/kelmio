-- +goose Up
ALTER TABLE issues
  ADD COLUMN IF NOT EXISTS archived_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_issues_archived_at ON issues(archived_at);

-- +goose Down
DROP INDEX IF EXISTS idx_issues_archived_at;

ALTER TABLE issues
  DROP COLUMN IF EXISTS archived_at;
