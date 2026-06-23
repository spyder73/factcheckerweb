-- +goose Up
-- +goose StatementBegin
CREATE TABLE stripe_customers (
  user_id            BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  stripe_customer_id TEXT         NOT NULL UNIQUE,
  created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE plan_events (
  id                BIGSERIAL    PRIMARY KEY,
  user_id           BIGINT       REFERENCES users(id) ON DELETE SET NULL,
  stripe_event_id   TEXT         NOT NULL UNIQUE,   -- idempotency: Stripe redelivers
  kind              TEXT         NOT NULL,          -- 'checkout.session.completed', 'customer.subscription.deleted', ...
  payload_json      JSONB        NOT NULL,
  processed_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX plan_events_user_id_idx ON plan_events (user_id);
CREATE INDEX plan_events_kind_idx    ON plan_events (kind);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS plan_events;
DROP TABLE IF EXISTS stripe_customers;
