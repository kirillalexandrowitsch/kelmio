-- +goose Up
CREATE TABLE IF NOT EXISTS saved_filters (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL,
  user_id uuid NOT NULL,
  name text NOT NULL,
  filters jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT saved_filters_workspace_user_fk
    FOREIGN KEY (workspace_id, user_id)
    REFERENCES workspace_members(workspace_id, user_id)
    ON DELETE CASCADE,
  CONSTRAINT saved_filters_name_not_blank CHECK (btrim(name) <> ''),
  CONSTRAINT saved_filters_filters_object CHECK (jsonb_typeof(filters) = 'object'),
  UNIQUE (workspace_id, user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_saved_filters_workspace_user
  ON saved_filters(workspace_id, user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_saved_filters_workspace_user;

DROP TABLE IF EXISTS saved_filters;
