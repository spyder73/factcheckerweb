-- +goose Up
-- +goose StatementBegin
CREATE TABLE checks (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         BIGINT       REFERENCES users(id) ON DELETE SET NULL,  -- NULL = anonymous
  anon_token      BYTEA,                              -- one-time SSE auth for anonymous checks (hashed)
  input_url       TEXT         NOT NULL,
  input_caption   TEXT         NOT NULL DEFAULT '',
  status          TEXT         NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','processing','completed','error','timeout')),
  plan_at_request TEXT         NOT NULL CHECK (plan_at_request IN ('anon','free','byok','plus','admin')),
  fanout_n        INTEGER      NOT NULL DEFAULT 1,
  byok_used       BOOLEAN      NOT NULL DEFAULT FALSE,
  byok_fellback   BOOLEAN      NOT NULL DEFAULT FALSE,  -- true if user requested BYOK but we fell back to pool
  error           TEXT,
  total_tokens_in INTEGER      NOT NULL DEFAULT 0,
  total_tokens_out INTEGER     NOT NULL DEFAULT 0,
  total_cost_micros INTEGER    NOT NULL DEFAULT 0,
  truncated       BOOLEAN      NOT NULL DEFAULT FALSE,  -- hit per-check deadline
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ
);
CREATE INDEX checks_user_id_idx     ON checks (user_id);
CREATE INDEX checks_created_at_idx  ON checks (created_at);
CREATE INDEX checks_status_idx      ON checks (status);
-- +goose StatementEnd

-- +goose StatementBegin
-- gen_random_uuid() needs pgcrypto on older Postgres; PG13+ has it built-in but require explicitly.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS checks;
