-- +goose Up
-- +goose StatementBegin
CREATE TABLE sources (
  id            BIGSERIAL    PRIMARY KEY,
  domain        TEXT         NOT NULL UNIQUE,    -- eTLD+1, lowercased, no scheme/path
  name          TEXT         NOT NULL,           -- human-readable outlet name
  category      TEXT         NOT NULL DEFAULT 'general'
                CHECK (category IN ('wire','newspaper','magazine','specialist','academic','government','factcheck','encyclopedia','primary','general')),
  country       TEXT,                              -- ISO 3166-1 alpha-2, optional
  trust_tier    TEXT         NOT NULL DEFAULT 'unknown'
                CHECK (trust_tier IN ('tier1','tier2','tier3','unknown')),
  vetting_notes TEXT,
  active        BOOLEAN      NOT NULL DEFAULT TRUE,
  added_by      BIGINT       REFERENCES users(id) ON DELETE SET NULL,
  added_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX sources_trust_tier_idx ON sources (trust_tier) WHERE active = TRUE;
CREATE INDEX sources_category_idx   ON sources (category)   WHERE active = TRUE;
CREATE INDEX sources_country_idx    ON sources (country)    WHERE active = TRUE;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS sources;
