-- +goose Up
CREATE INDEX IF NOT EXISTS idx_issues_project_created_id
  ON issues(project_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_issues_project_due_created_id
  ON issues(project_id, due_date ASC NULLS LAST, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_issues_project_priority_created_id
  ON issues(
    project_id,
    (CASE priority
      WHEN 'critical' THEN 4
      WHEN 'high' THEN 3
      WHEN 'medium' THEN 2
      WHEN 'low' THEN 1
      ELSE 0
    END) DESC,
    created_at DESC,
    id DESC
  );

CREATE INDEX IF NOT EXISTS idx_issues_sprint_status_created_id
  ON issues(sprint_id, status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_workspace_user_created_id
  ON notifications(workspace_id, user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread_created_id
  ON notifications(workspace_id, user_id, created_at DESC, id DESC)
  WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_activity_log_issue_created_id
  ON activity_log(entity_type, entity_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_issue_labels_label_issue
  ON issue_labels(label_id, issue_id);

-- +goose Down
DROP INDEX IF EXISTS idx_issue_labels_label_issue;
DROP INDEX IF EXISTS idx_activity_log_issue_created_id;
DROP INDEX IF EXISTS idx_notifications_unread_created_id;
DROP INDEX IF EXISTS idx_notifications_workspace_user_created_id;
DROP INDEX IF EXISTS idx_issues_sprint_status_created_id;
DROP INDEX IF EXISTS idx_issues_project_priority_created_id;
DROP INDEX IF EXISTS idx_issues_project_due_created_id;
DROP INDEX IF EXISTS idx_issues_project_created_id;
