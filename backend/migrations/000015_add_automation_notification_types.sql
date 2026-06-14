-- +goose Up
ALTER TABLE notifications
  DROP CONSTRAINT IF EXISTS notifications_notification_type_check;

ALTER TABLE notifications
  ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN (
      'issue_assigned',
      'issue_mentioned',
      'issue_commented',
      'issue_automation_assigned',
      'issue_automation_status_changed',
      'sprint_started',
      'sprint_completed'
    )
  );

-- +goose Down
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM notifications
    WHERE notification_type IN (
      'issue_automation_assigned',
      'issue_automation_status_changed'
    )
  ) THEN
    RAISE EXCEPTION 'cannot remove automation notification types while automation notifications exist';
  END IF;
END $$;

ALTER TABLE notifications
  DROP CONSTRAINT IF EXISTS notifications_notification_type_check;

ALTER TABLE notifications
  ADD CONSTRAINT notifications_notification_type_check CHECK (
    notification_type IN (
      'issue_assigned',
      'issue_mentioned',
      'issue_commented',
      'sprint_started',
      'sprint_completed'
    )
  );
