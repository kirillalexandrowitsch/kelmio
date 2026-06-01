-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  issue_id uuid REFERENCES issues(id) ON DELETE CASCADE,
  notification_type text NOT NULL CHECK (
    notification_type IN (
      'issue_assigned',
      'issue_mentioned',
      'issue_commented',
      'sprint_started',
      'sprint_completed'
    )
  ),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT notifications_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_notifications_workspace_user_created
  ON notifications(workspace_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread
  ON notifications(workspace_id, user_id, created_at DESC)
  WHERE read_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_workspace_user_created;

DROP TABLE IF EXISTS notifications;
