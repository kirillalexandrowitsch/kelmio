-- +goose Up
CREATE TABLE IF NOT EXISTS team_invites (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  email text NOT NULL,
  role text NOT NULL CHECK (role IN ('admin', 'member')),
  token_hash text NOT NULL UNIQUE,
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  accepted_at timestamptz,
  revoked_at timestamptz,
  CHECK (btrim(email) <> ''),
  CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idx_team_invites_workspace_id ON team_invites(workspace_id);
CREATE INDEX IF NOT EXISTS idx_team_invites_token_hash ON team_invites(token_hash);
CREATE INDEX IF NOT EXISTS idx_team_invites_expires_at ON team_invites(expires_at);
CREATE INDEX IF NOT EXISTS idx_team_invites_created_by ON team_invites(created_by);
CREATE UNIQUE INDEX IF NOT EXISTS idx_team_invites_pending_email
  ON team_invites(workspace_id, lower(email))
  WHERE accepted_at IS NULL AND revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS team_invites;
