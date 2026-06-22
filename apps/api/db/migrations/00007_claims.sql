-- +goose Up
-- +goose StatementBegin
CREATE TABLE claims (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  check_id      UUID         NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
  position      INTEGER      NOT NULL,                -- 0-indexed within the check
  raw_text      TEXT         NOT NULL,                -- the original claim text (what the user sees)
  canonical_text TEXT        NOT NULL,                -- normalized for caching: NFKC + lowercase + collapsed whitespace
  claim_hash    BYTEA        NOT NULL,                -- sha256(canonical_text) — used as the verdict cache key
  difficulty    TEXT         NOT NULL DEFAULT 'medium'
                CHECK (difficulty IN ('easy','medium','hard')),
  high_stakes   BOOLEAN      NOT NULL DEFAULT FALSE,  -- vaccines, elections, medical, etc — never early-exit
  cache_hit     BOOLEAN      NOT NULL DEFAULT FALSE,  -- served from Redis without running fanout
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX claims_check_id_idx   ON claims (check_id);
CREATE INDEX claims_claim_hash_idx ON claims (claim_hash);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS claims;
