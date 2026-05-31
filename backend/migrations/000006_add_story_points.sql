-- +goose Up
ALTER TABLE issues
  ADD COLUMN IF NOT EXISTS story_points integer NOT NULL DEFAULT 0
  CHECK (story_points >= 0 AND story_points <= 100);

CREATE INDEX IF NOT EXISTS idx_issues_sprint_story_points
  ON issues(sprint_id, story_points);

-- +goose Down
DROP INDEX IF EXISTS idx_issues_sprint_story_points;

ALTER TABLE issues
  DROP COLUMN IF EXISTS story_points;
