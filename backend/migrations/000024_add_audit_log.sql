-- +goose Up
-- Minimal administrative audit foundation (PLAT-006). Records who performed an
-- administrative action, on which target, within which organization. Kept
-- deliberately small; it is expanded in later versions. Additive and idempotent.

CREATE TABLE IF NOT EXISTS audit_log (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid REFERENCES organizations(id) ON DELETE SET NULL,
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  action text NOT NULL,
  target_type text NOT NULL,
  target_id uuid,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_organization
  ON audit_log(organization_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_log_organization;
DROP TABLE IF EXISTS audit_log;
