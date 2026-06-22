-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
  token_hash    BYTEA       PRIMARY KEY,        -- SHA-256 of the opaque cookie value
  user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  csrf_token    BYTEA       NOT NULL,            -- 32 random bytes; double-submit pattern
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at    TIMESTAMPTZ NOT NULL,
  last_used_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ip            INET,
  user_agent    TEXT
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX sessions_user_id_idx    ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS sessions;
