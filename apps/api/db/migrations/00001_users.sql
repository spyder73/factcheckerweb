-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE users (
  id                  BIGSERIAL PRIMARY KEY,
  email               CITEXT      NOT NULL UNIQUE,
  password_hash       TEXT        NOT NULL,
  email_verified_at   TIMESTAMPTZ,
  plan                TEXT        NOT NULL DEFAULT 'free'
                                  CHECK (plan IN ('free','byok','plus','admin')),
  totp_secret         TEXT,
  totp_enabled        BOOLEAN     NOT NULL DEFAULT FALSE,
  failed_login_count  INTEGER     NOT NULL DEFAULT 0,
  locked_until        TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX users_locked_idx ON users (locked_until) WHERE locked_until IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
