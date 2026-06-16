-- +goose Up
CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL,
  request_ip_hash text NOT NULL DEFAULT '',
  request_user_agent text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  revoked_at timestamptz,
  CONSTRAINT password_reset_tokens_hash_valid CHECK (token_hash ~ '^[a-f0-9]{64}$'),
  CONSTRAINT password_reset_tokens_request_ip_hash_valid CHECK (request_ip_hash = '' OR request_ip_hash ~ '^[a-f0-9]{64}$'),
  CONSTRAINT password_reset_tokens_user_agent_valid CHECK (char_length(request_user_agent) <= 240),
  CONSTRAINT password_reset_tokens_expires_after_created CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_password_reset_tokens_token_hash
  ON password_reset_tokens(token_hash);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_created
  ON password_reset_tokens(user_id, created_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_active_user
  ON password_reset_tokens(user_id, expires_at, id)
  WHERE used_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expiry
  ON password_reset_tokens(expires_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_password_reset_tokens_expiry;
DROP INDEX IF EXISTS idx_password_reset_tokens_active_user;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_created;
DROP INDEX IF EXISTS idx_password_reset_tokens_token_hash;
DROP TABLE IF EXISTS password_reset_tokens;
