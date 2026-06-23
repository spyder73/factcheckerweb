-- Phase 7 schema additions:
--   M3 — corrections chain. When we later learn a verdict was wrong (new
--   evidence, source retracted, our pipeline made a mistake), we issue a
--   NEW verdict and link it back via superseded_by. The old verdict is NOT
--   deleted — non-repudiation matters more than tidiness. UI shows
--   "Updated YYYY-MM-DD — newer verdict available" on the old page.
--
--   Retention windows. Anonymous checks get a 24h purge (M10 sketched).
--   We add the columns + an UNCOMMENTED partial index so a single cron
--   can scan + delete cheaply. Actual purge job lives in apps/api/cron/.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE verdicts
  ADD COLUMN superseded_by UUID REFERENCES verdicts(id) ON DELETE SET NULL,
  ADD COLUMN superseded_at TIMESTAMPTZ,
  ADD COLUMN correction_reason TEXT;
CREATE INDEX verdicts_superseded_by_idx ON verdicts (superseded_by)
  WHERE superseded_by IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Anonymous-check retention. anon_token is already present on checks;
-- this is the timestamp the purge job uses to decide what to drop.
-- 24h default per M10 from Phase A.
ALTER TABLE checks
  ADD COLUMN expires_at TIMESTAMPTZ;

-- Backfill: only anonymous (user_id IS NULL) rows get a 24h expiry from now.
UPDATE checks
   SET expires_at = NOW() + INTERVAL '24 hours'
 WHERE user_id IS NULL
   AND expires_at IS NULL;

-- Partial index so the purge job's scan is index-only.
CREATE INDEX checks_expires_at_idx ON checks (expires_at)
  WHERE expires_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS checks_expires_at_idx;
ALTER TABLE checks DROP COLUMN IF EXISTS expires_at;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS verdicts_superseded_by_idx;
ALTER TABLE verdicts
  DROP COLUMN IF EXISTS correction_reason,
  DROP COLUMN IF EXISTS superseded_at,
  DROP COLUMN IF EXISTS superseded_by;
-- +goose StatementEnd
