-- +goose Up
-- Reusable role assignments (PLAT-007). A single polymorphic table maps a
-- subject (a user or a group) to a role within a scope (a workspace or a
-- project). The effective role of a user is later resolved as the maximum of
-- their direct assignments and any group-derived assignments. scope_id and
-- subject_id are polymorphic, so they carry no foreign keys; referential
-- cleanup is handled in the application layer. Additive and idempotent.

CREATE TABLE IF NOT EXISTS role_assignments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  scope text NOT NULL CHECK (scope IN ('workspace', 'project')),
  scope_id uuid NOT NULL,
  subject_type text NOT NULL CHECK (subject_type IN ('user', 'group')),
  subject_id uuid NOT NULL,
  role text NOT NULL,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (scope, scope_id, subject_type, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_role_assignments_scope
  ON role_assignments(scope, scope_id);
CREATE INDEX IF NOT EXISTS idx_role_assignments_subject
  ON role_assignments(subject_type, subject_id);

-- +goose Down
DROP INDEX IF EXISTS idx_role_assignments_subject;
DROP INDEX IF EXISTS idx_role_assignments_scope;
DROP TABLE IF EXISTS role_assignments;
