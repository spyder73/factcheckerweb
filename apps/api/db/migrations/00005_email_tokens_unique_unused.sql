-- +goose Up
-- +goose StatementBegin
-- Defence in depth: only one unused token per (user_id, purpose) can exist.
-- The Forgot handler invalidates prior unused tokens in the same transaction
-- as the INSERT; this constraint guarantees that even a buggy code path can't
-- leave two valid reset tokens outstanding.
CREATE UNIQUE INDEX email_tokens_unique_unused
  ON email_tokens (user_id, purpose) WHERE used_at IS NULL;
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS email_tokens_unique_unused;
