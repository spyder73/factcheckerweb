# Phase A Findings — quick reference

Five-agent independent adversarial review (skeptical journalist, misinfo researcher, vulnerable-pop UX, privacy researcher, cost+scaling engineer) ran on 2026-06-23. Full reports + synthesis: [`INDEPENDENT_REVIEW.md`](INDEPENDENT_REVIEW.md). This file is the scan-in-30-seconds version.

---

## Three findings serious enough to pause and rethink

1. **The word "verified" is dishonest until a benchmark exists.** Four of five reviewers independently arrived at this. The system has no held-out adversarial benchmark with calibration data; calling its outputs "verified" inherits a trust contract the system cannot honor. Either earn the title (Phase 6 benchmark + named editorial board + IFCN aspiration) or reposition publicly as "evidence triage" / "claim research assistant". Current middle ground is not honest.

2. **The privacy posture is incompatible with the journalist persona we're courting.** Indefinite IP+UA retention on `audit_log`, indefinite URL retention on `checks`, scraper that leaks user-IP to fetched outlets, SSE timing that leaks verdict-class to network observers. A Hungarian or Turkish journalist using Alethea today is materially LESS safe than they would be using raw Google over Tor. Fix this before recruiting journalists or exclude them from the launch persona.

3. **Unit economics kill the project before the methodology questions become moot.** One viral news event can cost $20k–80k in an afternoon. Runway exhausts somewhere between 2,000 and 8,000 active free users with no cost infrastructure. Phase 4.5 (cost infra) is not optional, not parallelizable with launch — must ship first.

**Synthesis verdict:** delay public launch 4-8 weeks; reposition; execute Phase 4.5 + the human TODO items in parallel; treat the benchmark as the gating artifact for every subsequent verdict-quality claim.

---

## 24 MUST-FIX-BEFORE-PUBLIC-LAUNCH items

Categorised by where they land in the current phase plan.

### Phase 4.5 — design polish + verdict copy + cost infrastructure

| # | Finding | One-liner |
|---|---|---|
| M1 | Rename "verified", rewrite all verdict labels in plain language with action verbs on uncertain verdicts | "Don't share this" beats `unverifiable` |
| M4 | Per-user daily $ budget (not check-count) + global daily spend circuit-breaker on the AI provider | money cap, not call cap |
| M5 | Anthropic / OpenRouter prompt caching on the shared source pool | single largest cost reduction available |
| M6 | Semantic cache (embedding + ANN + 0.92 cosine) on top of exact-match cache | first viral claim breaks the budget without it |
| M7 | Hard cap images per check (4) + per-agent `max_tokens` + per-check token circuit breaker | bounds the prompt-injection-token-spew attack |
| M11 | Verdict pages `noindex,nofollow` by default; transparency-log publication opt-in | SEO-laundered "verified" badge is the worst case |
| M14 | Claim-class-aware cache TTL (political 1-6h, scientific 90d, historical 365d) | 30d for everything is too long for political claims |
| M17 | Replace numeric confidence ("0.62") with bands (high/medium/low) | numbers communicate false precision |
| M19 | Audit every error state; replace engineer-copy with action-oriented user guidance | refer to Snopes/regional fact-checkers during outages |
| M20 | Emit ClaimReview JSON-LD on every (opted-in) verdict page | table stakes for fact-check-ecosystem credibility |
| M23 | SSE timeline OFF by default (opt-in for power users); pad SSE events against timing inference | three reviewers said hide it |

### Phase 5 — billing + dispute + data-rights UX

| # | Finding |
|---|---|
| M3 (UI) | Dispute mechanism with visible "under review" + `superseded_by` versioning UI + correction history |
| M8 (UX) | Data export endpoint (GDPR Art. 15) + delete endpoint (Art. 17) + privacy notice naming IP/UA/claim retention |
| M10 | Stop calling guest checks "anonymous". Rename to "guest". Banner explaining what is/isn't retained. /24 IP truncation + 24h purge for guest |
| M15 | Refuse-to-verdict on languages where curated source coverage is below threshold |
| M18 | Disable share-images <24h old OR include timestamp + model+prompt version + live-page QR |

### Phase 6 — pipeline restructure + benchmark harness

| # | Finding |
|---|---|
| M2 | Pre-registered adversarial benchmark suite (300+ ground-truth claims, calibration, paraphrase stability, citation grounding) |
| M12 | "Do not check" refusal list: medical advice on named individuals, ongoing legal proceedings, defamatory-shape claims about private individuals |
| M13 | Date-aware retrieval: queries include temporal expressions; "X said Y today" must not pull 2019 sources |
| M16 | Two-pass extraction: cheap model strips scraped content to atomic claims, judge NEVER sees raw scraped prose |
| M22 | Citation grounding verifier as final stage: NLI-check each cited URL+claim against the actual paragraph; strip un-grounded; demote |

### Phase 7 — corrections backend + retention policies

| # | Finding |
|---|---|
| M3 (backend) | `superseded_by` chain, correction history table, dispute queue, SLA timers |

### Phase 8 — launch docs + Leichte Sprache

| # | Finding |
|---|---|
| M24 (DE) | German UI must pass *Leichte Sprache* review by a certified agency before German launch |

### Human TODO (non-code, blocks public launch)

| # | Finding |
|---|---|
| M2 (dataset) | Curate the 500-item adversarial benchmark; name a human evaluation lead. **Highest-priority non-code item.** |
| M8 (legal) | Retain EU-qualified Data Protection Officer or external DPO service |
| M9 | Commission GDPR Art. 35 DPIA from qualified counsel |
| M21 | Plus tier pricing decision — reviewers say $60-80+/mo minimum; $19 indefensible |
| M24 (translation) | Engage certified Leichte Sprache reviewer for German UI |

---

## 18 SHIP-BUT-MONITOR items

Acceptable to launch with provided we instrument early-warning signals. See [`INDEPENDENT_REVIEW.md`](INDEPENDENT_REVIEW.md) §"SHIP-BUT-MONITOR" for the table mapping each finding to its monitoring signal. Highlights:

- S1: Track verdict-flip rate under paraphrase; >5% means calibration is broken
- S2: Sample-audit `verified` verdicts whose sources have median first-seen <90 days (poisoning signal)
- S10: Track free→Plus conversion weekly; below 3% means reprice or restrict free tier
- S17: Re-run benchmark subset on every model bump; flag verdict flips publicly

## 30 V2-OR-LATER items

Real findings, not blocking. Roadmap. Highlights:

- V1: Genuinely diverse investigator architecture (different model families)
- V2: Structured-database adapters (Wikidata, OpenCorporates, CrossRef, Bundestag open data)
- V7: Public bug-bounty for verdict failures
- V8/V9: Tor support + decoy fetches for journalist threat model
- V10/V11: Quarterly transparency report + warrant canary
- V13: Model cascade (Haiku for cheap stages, Sonnet only where it matters)
- V20: Named editorial leadership; IFCN signatory status
- V22: Confidence intervals / Brier scores published alongside verdicts
- V23: "What would change this verdict" section forces falsifiability articulation
- V30: Two-surface UI — simple default + expert mode

---

## Where reviewers diverged (resolved positions)

- **Show non-expert users more vs. less detail?** → Both. Build progressive disclosure with a deliberate audience switch. Simple by default, expert mode one click away. (V30.)
- **Is BYOK the path forward?** → BYOK is the economic foundation (cost reviewer wins on substance) but the UX surface belongs in settings, not nav (UX reviewer wins on placement). Three distinct journeys: free (loss-leader trial), Plus (honestly priced for non-technical users), BYOK (power user with their own key).
- **Should live SSE timeline exist at all?** → Three reviewers said hide it; nobody defended it. Off by default, opt-in, padded against timing. (M23.)
- **How to handle the indefinite-retention problem?** → Reconcilable: verdict (content-addressed, not user-addressed) persists with versioning; user→check linkage aggressively expired. (M8, M10.)
- **Add multi-model fan-out?** → Yes; combine with cheap-first cascade. Cascade for non-investigator stages, model diversity for investigators. Net cost roughly neutral, epistemic quality up. (V1 + V13.)

---

## Reviewer-rejected ("NON-ISSUE") findings

Listed in `INDEPENDENT_REVIEW.md` Part 3 NON-ISSUE. Brief: drop free-tier entirely (no — keep as loss-leader, tighten instead), Sentry blanket ban (no — config the PII scrub), drop seven-state taxonomy entirely (no — collapse only at user-visible surface, keep richness in API + transparency log).
