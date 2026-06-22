-- +goose Up
-- +goose StatementBegin
CREATE TABLE journalist_applications (
  id              BIGSERIAL    PRIMARY KEY,
  user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  full_name       TEXT         NOT NULL,
  outlet          TEXT         NOT NULL,
  outlet_url      TEXT         NOT NULL,         -- main outlet domain
  byline_urls     TEXT[]       NOT NULL DEFAULT '{}',  -- 1..N URLs proving authorship
  country         TEXT,                            -- ISO 3166-1 alpha-2
  beat            TEXT,                            -- subject area e.g. "EU politics"
  bio             TEXT,
  status          TEXT         NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','approved','rejected','withdrawn')),
  reviewer_id     BIGINT       REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at     TIMESTAMPTZ,
  review_notes    TEXT,
  source_id       BIGINT       REFERENCES sources(id) ON DELETE SET NULL,  -- set on approval if outlet was promoted
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX journalist_apps_user_idx     ON journalist_applications (user_id);
CREATE INDEX journalist_apps_status_idx   ON journalist_applications (status) WHERE status = 'pending';
CREATE INDEX journalist_apps_created_idx  ON journalist_applications (created_at);

-- Each user has at most one pending application at a time. They can submit
-- a fresh one only after the prior is decided.
CREATE UNIQUE INDEX journalist_apps_one_pending_per_user
  ON journalist_applications (user_id) WHERE status = 'pending';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS journalist_applications;
