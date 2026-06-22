-- +goose Up
-- +goose StatementBegin
CREATE TABLE agent_runs (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  claim_id        UUID         NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
  role            TEXT         NOT NULL CHECK (role IN ('screener','retrieval','investigator','judge','intent')),
  style           TEXT,                                  -- 'empiricist','skeptic','historical' (investigators only)
  position        INTEGER      NOT NULL DEFAULT 0,       -- 0..N-1 for investigator role
  provider        TEXT         NOT NULL,                 -- 'mistral','openai','anthropic','openrouter'
  model           TEXT         NOT NULL,
  byok            BOOLEAN      NOT NULL DEFAULT FALSE,
  verdict         TEXT,                                  -- nullable for retrieval / intent which don't vote
  confidence      DOUBLE PRECISION,
  reasoning       TEXT,
  transcript_json JSONB,                                  -- full prompt+response pair for transparency
  tokens_in       INTEGER      NOT NULL DEFAULT 0,
  tokens_out      INTEGER      NOT NULL DEFAULT 0,
  cost_micros     INTEGER      NOT NULL DEFAULT 0,
  duration_ms     INTEGER      NOT NULL DEFAULT 0,
  errored         BOOLEAN      NOT NULL DEFAULT FALSE,
  error_message   TEXT,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX agent_runs_claim_id_idx ON agent_runs (claim_id);
CREATE INDEX agent_runs_role_idx     ON agent_runs (role);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE verdicts (
  id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  claim_id         UUID         NOT NULL UNIQUE REFERENCES claims(id) ON DELETE CASCADE,
  final_verdict    TEXT         NOT NULL,
  confidence       DOUBLE PRECISION NOT NULL,
  summary          TEXT         NOT NULL,
  judge_model      TEXT         NOT NULL,
  dissent_json     JSONB        NOT NULL,           -- [{position, style, verdict, confidence}, ...]
  skeptical_fallback BOOLEAN    NOT NULL DEFAULT FALSE,  -- true if confidence floor demoted to unverifiable
  pre_floor_verdict TEXT,                            -- what judge said before the floor
  pre_floor_confidence DOUBLE PRECISION,
  content_hash     BYTEA        NOT NULL,           -- sha256(prompt_version|canonical_claim|sorted(source_urls)|sorted(transcript_hashes)) — non-repudiation
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX verdicts_content_hash_idx ON verdicts (content_hash);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE citations (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  agent_run_id  UUID         NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
  source_id     BIGINT,                            -- references sources(id) once Phase 3 adds it (nullable for now)
  url           TEXT         NOT NULL,
  domain        TEXT         NOT NULL,
  title         TEXT,
  quote         TEXT,
  trust_tier    TEXT         NOT NULL DEFAULT 'unknown',
  stance        TEXT         CHECK (stance IS NULL OR stance IN ('supports','contradicts','neutral','context')),
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX citations_agent_run_id_idx ON citations (agent_run_id);
CREATE INDEX citations_domain_idx       ON citations (domain);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS citations;
DROP TABLE IF EXISTS verdicts;
DROP TABLE IF EXISTS agent_runs;
