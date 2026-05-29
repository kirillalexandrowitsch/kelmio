-- +goose Up
CREATE TABLE IF NOT EXISTS issue_links (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  target_issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
  link_type text NOT NULL CHECK (link_type IN ('blocks', 'relates')),
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT issue_links_not_self CHECK (source_issue_id <> target_issue_id),
  CONSTRAINT issue_links_unique_relation UNIQUE (source_issue_id, target_issue_id, link_type)
);

CREATE INDEX IF NOT EXISTS idx_issue_links_source_issue_id ON issue_links(source_issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_links_target_issue_id ON issue_links(target_issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_links_created_by ON issue_links(created_by);

CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_links_unique_relates_pair
  ON issue_links (
    LEAST(source_issue_id::text, target_issue_id::text),
    GREATEST(source_issue_id::text, target_issue_id::text)
  )
  WHERE link_type = 'relates';

-- +goose Down
DROP INDEX IF EXISTS idx_issue_links_unique_relates_pair;
DROP INDEX IF EXISTS idx_issue_links_created_by;
DROP INDEX IF EXISTS idx_issue_links_target_issue_id;
DROP INDEX IF EXISTS idx_issue_links_source_issue_id;

DROP TABLE IF EXISTS issue_links;
