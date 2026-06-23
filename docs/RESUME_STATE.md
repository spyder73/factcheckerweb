# RESUME STATE — read this first if context was just compacted

Last updated: 2026-06-23 during Phase 4.5 work, mid-context-compaction.

## Active goal (set by user via `/goal`)

> Finish everything: 5-agent independent brainstorm (Phase A, FIRST) → Phase 4.5 design polish + pipeline diagram → Phase 5 billing → Phase 6 mobile scaffolding → Phase 7 production deploy configs → Phase 8 launch docs. One commit per phase. Spawn workflows liberally. Final message MUST be the human-only TODO list (everything I have to do that you can't).

Goal stop-hook is active. Don't tell the user to clear it.

## Where we are RIGHT NOW

**Phase A** (5-agent brainstorm) — ✅ DONE + committed. Output saved to `docs/INDEPENDENT_REVIEW.md` (~150k chars, 5 charters + synthesis) and `docs/PHASE_A_FINDINGS.md` (scannable). PLAN.md updated to reflect the re-prioritization.

**Phase 4.5** — 🟡 IN PROGRESS, work-in-progress NOT YET COMMITTED. Files changed but uncommitted:
- `apps/web/src/components/PipelineDiagram.tsx` (NEW — custom SVG diagram, horizontal desktop + vertical mobile)
- `apps/web/src/hooks/useCountUp.ts` (NEW — intersection-observer triggered count-up)
- `apps/web/src/utils/head.tsx` (NEW — head manager for noindex + ClaimReview JSON-LD)
- `apps/web/src/design/tokens.ts` (verdict copy rewrite — M1 + M17 done)
- `apps/web/src/components/check/VerdictRing.tsx` (shows verdict name + confidence band, no numeric %)
- `apps/web/src/components/check/ResultBlock.tsx` (added action-line render)
- `apps/web/src/pages/Check.tsx` (useHead wired for noindex + ClaimReview)
- `apps/web/src/pages/CheckDetail.tsx` (useHead wired for noindex)
- `apps/web/src/pages/Landing.tsx` (uses PipelineDiagram + useCountUp)
- `apps/web/src/pages/HowItWorks.tsx` (uses PipelineDiagram explainer variant)

What's done in Phase 4.5:
- ✅ Custom SVG PipelineDiagram (landing + explainer variants)
- ✅ Count-up trust-strip animation on Landing
- ✅ M1: verdict labels rewritten (WELL-SUPPORTED / PARTLY TRUE / MISLEADING / CONTRADICTED / NOT ENOUGH EVIDENCE / LIKELY SATIRE / NO CHECKABLE CLAIM)
- ✅ M1: actionLine field on every verdict ("Don't share this without checking", etc.)
- ✅ M17: confidence bands (high/medium/low) instead of numeric percentage on VerdictRing
- ✅ M11: noindex,nofollow on /check and /check/{id}
- ✅ M20: ClaimReview JSON-LD emitted on /check when a verdict lands

What's left for Phase 4.5:
- [ ] Verify build is still clean after the verdict-copy + useHead changes
- [ ] M23: SSE timeline off-by-default with "Show details" toggle on /check
- [ ] Design polish: animated mesh-gradient body background (CSS @property + slow drift, ~40s, respects reduced-motion)
- [ ] Design polish: cursor-follow hero spotlight (Landing only, hero area, ~200ms lerp)
- [ ] Design polish: hero headline character-stagger fade-in on load
- [ ] Design polish: View Transitions API for route changes
- [ ] Design polish: verdict ring halo (narrative-tier only)
- [ ] M4: per-user daily $ budget + global circuit breaker (Go side, services/factcheck/budget.go)
- [ ] M7 (extension): hard cap images per check (4 not 6), per-agent max_tokens cap (probably already in code but tighten)
- [ ] Commit Phase 4.5 with one descriptive message
- [ ] (Optional) UX/design review workflow before commit — only if budget remains

What's deferred from Phase 4.5 to later phases or never:
- M5 (prompt caching) — needs Anthropic-specific message-block format; defer to Phase 6
- M6 (semantic cache) — needs embedding stack; defer to v2
- M14 (claim-class-aware cache TTL) — needs classifier; defer to Phase 6
- M19 (error states) — already mostly done in Phase 4

## Phases not yet started

**Phase 5** — Billing (Stripe + donations + dispute mechanism + GDPR data-rights endpoints):
Per the existing Phase 5 plan PLUS the M-items from Phase A:
- M3 (UI): dispute mechanism + "under review" + superseded_by versioning UI
- M8 (UX): /api/me/data-export + /api/me/data-delete endpoints (GDPR Art. 15 / 17)
- M10: rename anonymous → guest; banner; /24 IP truncation + 24h purge
- M15: refuse-to-verdict on languages where curated source coverage is below threshold
- M18: share-image timestamps + version + QR

**Phase 6** — Mobile scaffolding + pipeline-hardening from Phase A:
- Original: apps/mobile/ Expo scaffolding, packages/shared-types/, README
- Added M-items: M2 (benchmark harness skeleton), M12 (refuse list), M13 (date-aware retrieval), M16 (two-pass extraction for prompt-injection defense), M22 (citation grounding NLI verifier)

**Phase 7** — Production deploy configs (no actual deploy):
- infra/docker-compose.prod.yml
- infra/Caddyfile with auto-TLS, HSTS, security headers
- infra/backup/backup.sh (pg_dump | age | upload)
- infra/systemd/alethea.service
- infra/monitoring/uptime-kuma.yml
- docs/DEPLOY.md
- Plus M3 (backend): superseded_by chain, correction-history table, dispute queue, SLA timers
- Plus retention-policy migrations (audit_log IP truncation, /checks anon purge)

**Phase 8** — Launch docs:
- docs/TOS.md, docs/privacy.md, docs/security.md (all "LEGAL REVIEW REQUIRED")
- docs/how-it-works.md, docs/trust-policy.md, docs/source-policy.md, docs/journalist-program.md
- docs/CONTRIBUTING.md, docs/CHANGELOG.md
- /.well-known/security.txt served by the web nginx
- README.md polish
- Plus M24 (DE): note Leichte Sprache review required before DE launch

## Final report (MUST be the last message)

Human-only TODO list organised by category. Reference `docs/PHASE_A_FINDINGS.md` "Human TODO" section for the must-include items.

## Important context to remember

- AI_PROVIDER=openrouter is the user's current default. Don't break the Mistral path.
- The user is on macOS, uses `make dev` (Docker compose) to run.
- Bundle is React + Vite + TS + Tailwind + framer-motion + radix-ui + react-i18next.
- The user has explicitly said they want the page to "feel like a $10k design" — current background gradients (cobalt blob + dot grid) ARE now visible, user confirmed via screenshot.
- Phase A reviewers were UNANIMOUS that the live SSE timeline should be off by default; three reviewers, three lenses. Don't soft-pedal this in Phase 4.5.
- Goal stop-hook auto-clears when ALL phases (incl. final TODO list) are done. Don't tell user to clear.
- One commit per phase. Don't squash.
- If a workflow fails (any verify/refute agent), proceed with what you have — don't block.

## Critical files to NOT break

- `apps/api/services/ai/factory.go` — AI_PROVIDER override is wired
- `apps/api/services/factcheck/pipeline.go` — N investigator fanout, judge, cache
- `apps/api/ratelimit/ratelimit.go` — anon/free/byok/plus/admin caps with env overrides
- `apps/web/src/design/tokens.ts` — VERDICTS metadata is consumed by ~6 components
- `apps/web/src/main.tsx` — provider order: ThemeProvider > AuthProvider > RouterProvider
- `apps/web/src/i18n/en.json` — verdict copy keys still use the OLD labels; update to match new VERDICTS metadata at some point (currently the components read from VERDICTS directly, not via i18n keys, so this is decoupled but inconsistent)

## Token-budget remaining

Started this session with the goal; have already burned a chunk on Phase A workflow (~$50) + Phase 4.5 design workflow planning + this Phase 4.5 implementation. Stay efficient — Phase 5/6/7/8 are coming.

## How to resume

1. Read this file.
2. `cd /Users/dorian/Alethea && git status` — see Phase 4.5 uncommitted state.
3. `cd apps/web && npm run build` — confirm build clean.
4. Pick up from "What's left for Phase 4.5" above.
5. After committing Phase 4.5, proceed sequentially through Phases 5 → 6 → 7 → 8.
6. Final message MUST be the human TODO list — see PHASE_A_FINDINGS.md "Human TODO" + PHASE 8 doc work for the inputs.
