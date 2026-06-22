-- +goose Up
-- +goose StatementBegin
CREATE TABLE byok_keys (
  id            BIGSERIAL    PRIMARY KEY,
  user_id       BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider      TEXT         NOT NULL CHECK (provider IN ('mistral','openai','anthropic','openrouter')),
  label         TEXT         NOT NULL DEFAULT '',
  ciphertext    BYTEA        NOT NULL,     -- AES-GCM output of the user's key
  nonce         BYTEA        NOT NULL,     -- 12-byte GCM nonce
  last_used_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  -- One active key per (user, provider). Replacing a key is a DELETE+INSERT or UPDATE.
  UNIQUE (user_id, provider)
);
CREATE INDEX byok_keys_user_id_idx ON byok_keys (user_id);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS byok_keys;
