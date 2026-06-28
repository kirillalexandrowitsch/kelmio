-- +goose Up
-- Session-scoped active workspace for multi-workspace navigation (PLAT-006).
-- Nullable: when unset, the session resolves to the user's first membership,
-- preserving the existing single-workspace behavior.
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS active_workspace_id uuid REFERENCES workspaces(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE sessions DROP COLUMN IF EXISTS active_workspace_id;
