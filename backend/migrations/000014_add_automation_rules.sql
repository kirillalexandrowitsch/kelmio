-- +goose Up
CREATE TABLE IF NOT EXISTS automation_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name text NOT NULL,
  trigger_type text NOT NULL,
  conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
  actions jsonb NOT NULL,
  position integer NOT NULL,
  is_enabled boolean NOT NULL DEFAULT true,
  disabled_reason text,
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT automation_rules_name_valid CHECK (
    btrim(name) <> ''
    AND char_length(name) <= 100
  ),
  CONSTRAINT automation_rules_trigger_type_valid CHECK (
    trigger_type IN (
      'issue_created',
      'status_changed',
      'assignee_changed',
      'priority_changed'
    )
  ),
  CONSTRAINT automation_rules_conditions_valid CHECK (
    jsonb_typeof(conditions) = 'array'
    AND jsonb_array_length(conditions) <= 20
  ),
  CONSTRAINT automation_rules_actions_valid CHECK (
    jsonb_typeof(actions) = 'array'
    AND jsonb_array_length(actions) BETWEEN 1 AND 20
  ),
  CONSTRAINT automation_rules_position_valid CHECK (position > 0)
);

CREATE INDEX IF NOT EXISTS idx_automation_rules_project_position
  ON automation_rules(project_id, position, id);

CREATE INDEX IF NOT EXISTS idx_automation_rules_enabled_trigger
  ON automation_rules(project_id, trigger_type, position, id)
  WHERE is_enabled = true;

CREATE INDEX IF NOT EXISTS idx_automation_rules_created_by
  ON automation_rules(created_by, project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_automation_rules_created_by;
DROP INDEX IF EXISTS idx_automation_rules_enabled_trigger;
DROP INDEX IF EXISTS idx_automation_rules_project_position;
DROP TABLE IF EXISTS automation_rules;
