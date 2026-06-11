-- +goose Up
DROP TRIGGER IF EXISTS issues_sync_legacy_workflow_status ON issues;
DROP FUNCTION IF EXISTS sync_issue_legacy_workflow_status();

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_status_check;

ALTER TABLE issues
  ADD CONSTRAINT issues_status_key_valid CHECK (
    status ~ '^[a-z][a-z0-9_]{0,31}$'
  );

CREATE OR REPLACE FUNCTION sync_issue_workflow_status()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  resolved_status project_workflow_statuses%ROWTYPE;
BEGIN
  IF (TG_OP = 'INSERT' AND NEW.workflow_status_id IS NOT NULL)
    OR (TG_OP = 'UPDATE' AND NEW.workflow_status_id IS DISTINCT FROM OLD.workflow_status_id) THEN
    SELECT workflow_status.*
    INTO resolved_status
    FROM project_workflow_statuses workflow_status
    WHERE workflow_status.id = NEW.workflow_status_id
      AND workflow_status.project_id = NEW.project_id
      AND workflow_status.archived_at IS NULL;
  ELSIF TG_OP = 'INSERT'
    OR NEW.status IS DISTINCT FROM OLD.status THEN
    SELECT workflow_status.*
    INTO resolved_status
    FROM project_workflow_statuses workflow_status
    WHERE workflow_status.project_id = NEW.project_id
      AND workflow_status.key = NEW.status
      AND workflow_status.archived_at IS NULL;
  ELSE
    RETURN NEW;
  END IF;

  IF resolved_status.id IS NULL THEN
    RAISE EXCEPTION
      'active workflow status was not found for project %',
      NEW.project_id
      USING ERRCODE = '23514';
  END IF;

  NEW.workflow_status_id := resolved_status.id;
  NEW.status := resolved_status.key;
  RETURN NEW;
END;
$$;

CREATE TRIGGER issues_sync_workflow_status
BEFORE INSERT OR UPDATE OF project_id, status, workflow_status_id ON issues
FOR EACH ROW
EXECUTE FUNCTION sync_issue_workflow_status();

-- +goose Down
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM issues
    WHERE status NOT IN ('backlog', 'todo', 'in_progress', 'blocked', 'done')
  ) THEN
    RAISE EXCEPTION
      'cannot rollback workflow issue statuses while custom status keys are in use';
  END IF;
END;
$$;

DROP TRIGGER IF EXISTS issues_sync_workflow_status ON issues;
DROP FUNCTION IF EXISTS sync_issue_workflow_status();

ALTER TABLE issues
  DROP CONSTRAINT IF EXISTS issues_status_key_valid;

ALTER TABLE issues
  ADD CONSTRAINT issues_status_check CHECK (
    status IN ('backlog', 'todo', 'in_progress', 'blocked', 'done')
  );

CREATE OR REPLACE FUNCTION sync_issue_legacy_workflow_status()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  SELECT workflow_status.id
  INTO NEW.workflow_status_id
  FROM project_workflow_statuses workflow_status
  WHERE workflow_status.project_id = NEW.project_id
    AND workflow_status.key = NEW.status
    AND workflow_status.archived_at IS NULL;

  IF NEW.workflow_status_id IS NULL THEN
    RAISE EXCEPTION
      'active workflow status with key "%" was not found for project %',
      NEW.status,
      NEW.project_id
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER issues_sync_legacy_workflow_status
BEFORE INSERT OR UPDATE ON issues
FOR EACH ROW
EXECUTE FUNCTION sync_issue_legacy_workflow_status();
