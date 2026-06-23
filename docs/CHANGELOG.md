# Changelog

Notable changes to Alethea. Format loosely follows [Keep a Changelog].
Dates are ISO-8601 UTC.

[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/

## [Unreleased]

Everything below is on `main` but not yet tagged. v0.1.0 will be tagged
once we're publicly launchable.

### Phase 8 — Launch docs (2026-06-22)

- Wrote `docs/{how-it-works,trust-policy,source-policy,journalist-program}.md`
  for users.
- Wrote `docs/{TOS,privacy,security}.md` — all marked **LEGAL REVIEW
  REQUIRED** because they capture intent but need a lawyer before
  going live.
- Wrote `docs/CONTRIBUTING.md` and this `CHANGELOG.md`.
- Polished `README.md`.
- Added `/.well-known/security.txt` via Caddy.

### Phase 7 — Production deploy configs (2026-06-22)

- `infra/docker-compose.prod.yml`, `infra/Caddyfile`, systemd units.
- Encrypted nightly backup script (age + rsync/S3).
- `docs/DEPLOY.md` runbook.
- M3 schema: `verdicts.superseded_by` correction chain.
- M10 schema: `checks.expires_at` for 24h anonymous-check retention.
- `/.well-known/security.txt`.

### Phase 6 — Mobile scaffolding + pipeline wins (2026-06-22)

- Expo Router app at `apps/mobile/` with 5 stub screens, share-intent
  activation rules, EAS profiles.
- `packages/shared-types/` is real — canonical TS types shared
  between web + mobile.
- Pipeline wins from Phase A:
  - **M12** refuse list: screener classifies inputs as out-of-scope
    (opinion, prediction, personal experience, etc.) and the pipeline
    short-circuits to a clearly-labeled refusal.
  - **M13** date-aware retrieval: screener prompt now includes the
    current ISO date so "today" / "this year" resolve correctly.

### Phase 5 — Billing + GDPR (2026-06-22)

- Stripe Checkout subscription mode + Customer Portal.
- Signed-webhook handler with idempotent processing via
  `plan_events.stripe_event_id UNIQUE`.
- `/api/economics` (real numbers replace the static ticker fallback).
- M8: `/api/me/data-export` (GDPR Art. 15) +
  `/api/me/data-delete` (Art. 17, with `{"confirmation":"DELETE"}`).
- Pricing CTA wired to Stripe; Plus users get a "Manage subscription"
  → Customer Portal button.

### Phase 4.5 — Design polish + Phase A quick wins (2026-06-21)

- Custom SVG pipeline diagram (no mermaid) with two variants —
  landing (decorative) + explainer (hover tooltips).
- M1 verdict-copy rewrite: "VERIFIED" → "WELL-SUPPORTED",
  "FALSE" → "CONTRADICTED", action lines on every verdict
  ("Don't share this — sources contradict it.").
- M17: confidence bands (high / medium / low) replace numeric %.
- M20: ClaimReview JSON-LD on verdict pages.
- M11: noindex meta on user verdict pages.
- M23: SSE timeline off by default; quiet status line with "Show
  details" toggle.
- M4: cost circuit-breakers — per-user 24h cap + platform-wide
  kill switch.
- Animated mesh background (42s drift, prefers-reduced-motion
  respected).

### Phase A — 5-agent independent brainstorm (2026-06-21)

- `docs/INDEPENDENT_REVIEW.md` — full 5-charter reports + synthesis
  (~150k chars).
- `docs/PHASE_A_FINDINGS.md` — categorized punch list.
- Found: 24 MUST-FIX (M1–M24), 18 SHIP-BUT-MONITOR, 30 V2-OR-LATER,
  7 NON-ISSUE. Re-prioritized the phase plan accordingly.

### Phase 4 — Web redesign (2026-06-21)

- Clinical-modern design system: tokens.ts, dark + light themes,
  View Transitions API, framer-motion accents.
- Pages: Landing, Check, CheckDetail, History, Login, Signup, Forgot,
  Reset, HowItWorks, Sources, Pricing, SettingsKeys,
  SettingsJournalist.
- EconomicsTicker, EvidenceMatrix, VerdictRing, ConfidenceMeter.
- i18n: EN canonical, DE translated, ES + FR stubs.

### Phase 3 — Curated sources + Journalist program (2026-06-20)

- `sources` table + Phase 3 admin CRUD endpoints + seed list.
- Trust-tier rerank in retrieval.
- `journalist_applications` table + apply / mine / withdraw + admin
  list / decide endpoints.

### Phase 2 — Multi-agent pipeline v2 (2026-06-20)

- Shared-retrieval + N-investigator fan-out + judge.
- Screener, retrieval, investigator, judge files in
  `apps/api/services/factcheck/`.
- Verdict cache (Redis), source pool dedup, dissent persistence.
- BYOK vault (AES-GCM with AAD on `(user_id, provider)`).
- SSE streaming via the `checkstream` hub.
- Skeptical floor at confidence < 0.65.
- Content hash for non-repudiation.

### Phase 1 — Postgres + Auth + Rate limiting (2026-06-19)

- Postgres schema (users, sessions, email_tokens, audit_log, checks,
  claims, agent_runs, verdicts, citations).
- Argon2id signup/login/logout/forgot/reset/verify, opaque session
  cookies, CSRF double-submit.
- Redis sliding-window rate limiter with per-tier env overrides.
- SSRF guard with DNS-resolve-then-recheck-at-connect.

### Phase 0 — Monorepo + dev infra (2026-06-19)

- Restructured the existing PoC into `apps/api`, `apps/web`,
  `services/scrapers/instagram`, `packages/shared-types`.
- `docker-compose.dev.yml`, Makefile, `.env.example`.

---

## [0.0.0-poc] — pre-history

The original single-agent PoC at `spyder73/fact-checker` was the
starting point for this rewrite. See its git history for that work.
