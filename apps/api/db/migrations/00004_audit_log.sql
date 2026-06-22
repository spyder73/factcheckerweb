-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_log (
  id           BIGSERIAL    PRIMARY KEY,
  user_id      BIGINT       REFERENCES users(id) ON DELETE SET NULL,
  action       TEXT         NOT NULL,            -- e.g. 'auth.login.success', 'auth.signup', 'auth.login.fail'
  target_kind  TEXT,                              -- e.g. 'user','session','byok_key'
  target_id    TEXT,
  ip           INET,
  user_agent   TEXT,
  metadata     JSONB,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX audit_log_user_id_idx    ON audit_log (user_id);
CREATE INDEX audit_log_action_idx     ON audit_log (action);
CREATE INDEX audit_log_created_at_idx ON audit_log (created_at);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS audit_log;
