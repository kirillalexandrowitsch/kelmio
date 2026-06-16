-- +goose Up
CREATE TABLE IF NOT EXISTS email_outbox (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid REFERENCES workspaces(id) ON DELETE SET NULL,
  email_type text NOT NULL,
  recipient_email text NOT NULL,
  template_data jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL DEFAULT 'pending',
  attempt_count integer NOT NULL DEFAULT 0,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  last_error text,
  deduplication_key text NOT NULL DEFAULT '',
  processing_started_at timestamptz,
  sent_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT email_outbox_email_type_valid CHECK (
    btrim(email_type) <> ''
    AND char_length(email_type) <= 80
  ),
  CONSTRAINT email_outbox_recipient_email_valid CHECK (
    btrim(recipient_email) <> ''
    AND char_length(recipient_email) <= 320
  ),
  CONSTRAINT email_outbox_template_data_valid CHECK (jsonb_typeof(template_data) = 'object'),
  CONSTRAINT email_outbox_status_valid CHECK (status IN ('pending', 'processing', 'sent', 'failed')),
  CONSTRAINT email_outbox_attempt_count_valid CHECK (attempt_count >= 0),
  CONSTRAINT email_outbox_deduplication_key_valid CHECK (char_length(deduplication_key) <= 160),
  CONSTRAINT email_outbox_processing_started_at_valid CHECK (
    (status = 'processing' AND processing_started_at IS NOT NULL)
    OR (status <> 'processing')
  ),
  CONSTRAINT email_outbox_sent_at_valid CHECK (
    (status = 'sent' AND sent_at IS NOT NULL)
    OR (status <> 'sent')
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_email_outbox_deduplication_key
  ON email_outbox(deduplication_key)
  WHERE deduplication_key <> '';

CREATE INDEX IF NOT EXISTS idx_email_outbox_claim
  ON email_outbox(status, next_attempt_at, created_at, id)
  WHERE status IN ('pending', 'processing');

CREATE INDEX IF NOT EXISTS idx_email_outbox_stale_processing
  ON email_outbox(processing_started_at, id)
  WHERE status = 'processing';

CREATE INDEX IF NOT EXISTS idx_email_outbox_workspace_status
  ON email_outbox(workspace_id, status, created_at DESC, id);

-- +goose Down
DROP INDEX IF EXISTS idx_email_outbox_workspace_status;
DROP INDEX IF EXISTS idx_email_outbox_stale_processing;
DROP INDEX IF EXISTS idx_email_outbox_claim;
DROP INDEX IF EXISTS idx_email_outbox_deduplication_key;
DROP TABLE IF EXISTS email_outbox;
