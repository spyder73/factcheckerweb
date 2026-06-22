-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_tokens (
  token_hash  BYTEA       PRIMARY KEY,           -- SHA-256 of the single-use token
  user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose     TEXT        NOT NULL CHECK (purpose IN ('verify_email','reset_password')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at  TIMESTAMPTZ NOT NULL,
  used_at     TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX email_tokens_user_id_idx  ON email_tokens (user_id);
CREATE INDEX email_tokens_expires_idx  ON email_tokens (expires_at);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS email_tokens;
