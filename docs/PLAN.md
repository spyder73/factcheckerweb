# Alethea — Build Plan

This document is the durable source of truth for Alethea's build. Update it as decisions land.

## 0. Product north star

**Promise to the user:** *paste anything, get a verdict you can trace.* Every claim shows:

- The verdict + a confidence score
- Which agents voted which way (when multi-agent)
- Every source cited, with its trust tier
- Why the system is skeptical of itself when evidence is thin
- A separate "intent" panel — what the post is *trying* to do (sell, persuade, mock, inform) — clearly labeled as interpretation, not fact

**Three non-negotiables that protect trust:**

1. **Skeptical default** — when evidence is thin, the verdict is `Unverifiable`, never a guess dressed up as `True/False`. Confidence < threshold ⇒ system explicitly says "we don't know."
2. **Show your work** — every verdict expandable into agent transcripts and source quotes. No black box.
3. **Open source, reproducible** — anyone can run the same check and see the same logic.

The Swift app on `spyder73/fact-checker` — retire it once Expo iOS works. Maintaining two iOS clients solo is a trap; the Swift code can serve as a reference for any iOS-specific polish later.

---

## 1. Business model

Reality check on willingness-to-pay: fact-checking falls in the mental bucket of "stuff that should be free." Realistic ARPU is low. Strategy assumes that and uses cost discipline + BYOK to keep the free tier sustainable.

### Tiers

| Tier | Price | Who | What they get |
| --- | --- | --- | --- |
| Anonymous | Free | Casual visitor | 3 checks / day / IP, single cheap model, hard caps, no history |
| Free account | Free | Signed in | 10 checks / day, history, cheap model, longer cache TTL |
| BYOK | Free | Power users / journalists / privacy-conscious | Unlimited (capped only by their provider), choice of provider+model, key encrypted at rest |
| Alethea Plus | €5/mo or €45/yr | Believers / heavy users | Multi-agent N=5 consensus, strong judge model, priority queue, higher rate limits, source filters, exportable PDFs |
| Donations | Any | Supporters | Open Collective + GitHub Sponsors, transparent monthly cost page |
| API / NGO tier | Custom | Newsrooms, researchers | Server-to-server API, audit log export, higher limits — open later |

### Cost-control levers wired into the architecture

- **Cascade**: cheap screener model first (Mistral Small / Haiku) → escalate to strong model only on uncertainty
- **Verdict cache**: hash(normalized_claim) → verdict, 7d TTL for trending topics, 90d for stable facts
- **Image budget**: free max 3 images per check, Plus max 8
- **Token caps**: hard per-check ceiling that returns "Unverifiable — over budget" rather than silently degrading
- **Search budget**: per-tier search-call cap (Tavily/Brave/Serper)
- **No video transcription** in v1; Plus-only when added

### Transparent finances page

`/economics` shows this month's compute + search + scrape spend and donation/subscription revenue. Doubles as marketing — radical transparency builds the trust verdicts depend on.

---

## 2. Architecture

```
/Users/dorian/Alethea/
├── apps/
│   ├── api/                  ← evolved Go backend (module: alethea/api)
│   ├── web/                  ← React + Vite (v1); migrate to Expo Router web later
│   └── mobile/               ← Phase 6: Expo (RN) for iOS + Android
├── packages/
│   ├── shared-types/         ← TS types generated from Go structs (oapi-codegen, Phase 1+)
│   └── ui-tokens/            ← Phase 4: color/spacing tokens shared web↔mobile
├── services/
│   └── scrapers/
│       └── instagram/        ← Python service, dockerized, internal-only
├── infra/
│   ├── docker-compose.dev.yml
│   ├── docker-compose.prod.yml (Phase 7)
│   ├── Caddyfile             (Phase 7)
│   ├── migrations/           ← goose SQL migrations (Phase 1+)
│   └── backup/               ← pg_dump cron script (Phase 7)
└── docs/
    ├── PLAN.md               (this file)
    ├── how-it-works.md       (Phase 8)
    ├── trust-policy.md
    ├── source-policy.md
    ├── journalist-program.md
    ├── TOS.md
    ├── privacy.md
    └── security.md
```

### Backend evolution path (no rewrite)

The existing Mistral pipeline stays, gets wrapped. New backend shape:

```
apps/api/
  services/
    ai/        keep + add: openai, anthropic, google, openrouter
    scrapers/  keep + add: twitter (nitter fallback), tiktok, youtube
    search/    NEW: tavily, brave, serper providers behind an interface
    pipeline/  NEW: stages — extract → fanout → judge — replaces factcheck.go internals
    sources/   NEW: curated source registry, trust scoring
    cache/     NEW: claim-hash verdict cache (redis)
    byok/      NEW: encrypted key vault per user
  auth/        NEW: signup, login, sessions, csrf
  billing/     NEW: stripe webhooks, plan state
  admin/       NEW: journalist queue, source review
  ratelimit/   NEW: middleware
  db/          NEW: sqlc generated, migrations
```

### Multi-agent pipeline shape

```
URL/text
  │
  ▼
[Scraper]  →  raw content + media
  │
  ▼
[Claim Extractor — cheap model]  →  list of atomic checkable claims
  │
  ▼  (fan-out, parallel, per claim)
[Investigator × N]   ← N=1 free, N=3 BYOK default, N=5 Plus
  │  each agent: search → read → write {verdict, confidence, citations, reasoning}
  ▼
[Judge — strongest model]
  │  inputs: all N investigator outputs + their citations
  │  outputs: final verdict, agreement metric, dissent notes,
  │           skeptical fallback if confidence < threshold
  ▼
[Intent analyzer]  →  labeled "interpretation" — not fact
  │
  ▼
Persist + stream to client via SSE
```

**Prompt-injection mitigation:** scraped content is wrapped in a clearly delimited untrusted-data block (`<<<CONTENT>>>...<<</CONTENT>>>`) with explicit instruction to treat as data, not instructions. Never put user URL/content in the system prompt.

---

## 3. Database schema (sketch)

```
users              (id, email, password_hash, email_verified_at, plan, created_at, ...)
sessions           (token_hash, user_id, expires_at, last_used_at, ip, user_agent)
byok_keys          (id, user_id, provider, ciphertext, nonce, label, last_used_at)
checks             (id, user_id|null, input_url, input_text, status, created_at)
claims             (id, check_id, text, position)
agent_runs         (id, claim_id, model, provider, verdict, confidence, reasoning,
                    tokens_in, tokens_out, cost_micros)
citations          (id, agent_run_id, source_id|null, url, title, quote, trust_tier)
verdicts           (id, claim_id, final_verdict, confidence, judge_model, dissent_summary,
                    content_hash) -- content_hash anchors the verdict transparency log
sources            (id, domain, name, category, country, trust_tier, vetting_notes,
                    added_by, added_at)
journalist_apps    (id, user_id, name, outlet, urls, country, status, reviewer_id,
                    reviewed_at, notes)
audit_log          (id, user_id|null, action, target_kind, target_id, ip, ua, created_at)
plan_events        (id, user_id, stripe_event_id, kind, payload_json, processed_at)
rate_buckets       (key, count, window_start)  -- or redis
abuse_reports      (id, check_id, reporter_user|ip, reason, status)
```

`cost_micros` (integer microcents) makes the /economics page easy.

---

## 4. Security plan

### Wired into the backend from day 1

- **TLS everywhere via Caddy** — automatic ACME, HSTS preload
- **Cookies**: `HttpOnly`, `Secure`, `SameSite=Lax`, opaque tokens (not JWT — easier to revoke)
- **CSRF**: double-submit token for any mutating request
- **Passwords**: argon2id with sane params, never logged, never echoed
- **Sessions**: rotate on privilege change, max 30d, idle timeout 7d
- **Rate limiting**: token-bucket per (IP, endpoint) for anon, per (user, endpoint) for auth, stricter on `/auth/*` and `/check`
- **Trusted-proxy aware client-IP**: `X-Forwarded-For` is honored ONLY when `r.RemoteAddr` is in the configured `TRUSTED_PROXIES` CIDR list. Empty list = always use the connection peer (anti-spoof default for direct exposure). Behind Caddy: set `TRUSTED_PROXIES=127.0.0.1/32,::1/128,172.16.0.0/12` and rely on Caddy's default XFF-overwrite behavior.
- **SSRF defense on user-supplied URLs**: critical because the scraper fetches arbitrary URLs. Resolve DNS once, reject private/loopback/link-local/IPv6-ULA/cloud-metadata IPs (169.254.169.254, fd00::/8, 10/8, 172.16/12, 192.168/16, 127/8). Custom `DialContext` that re-checks the resolved IP at connect time (TOCTOU). Only http/https schemes, max body size, max redirects (3), strict timeout.
- **URL allowlist**: in addition to SSRF block, scope by platform: instagram.com, x.com/twitter.com, tiktok.com, facebook.com, youtube.com, plus generic HTTPS with the SSRF guard. Reject everything else with a clear message.
- **Prompt injection mitigation**: scraped text wrapped in `<<<CONTENT>>>...<<</CONTENT>>>` blocks with explicit instruction to treat as untrusted; never put user URL/content in system prompt; strip null bytes; cap length.
- **Output rendering**: markdown only, no raw HTML, no auto-executing links — render with `noopener noreferrer`; "leaving Alethea" interstitial for outbound clicks on Plus tier.
- **BYOK secrets**: AES-GCM (or nacl/secretbox) per-key, master key from `BYOK_MASTER_KEY` env (mounted file outside DB backup path, rotated quarterly). Never log keys; redact in error envelopes via a slog handler.
- **DB**: only Go service connects; not exposed; sqlc-generated parameterized queries; least-privilege role for app vs migrations.
- **Backups**: nightly `pg_dump` → encrypted with `age`, pushed to a Hetzner Storage Box (off-box, different credentials).
- **Audit log**: every login, key add/remove, journalist approve/reject, source add/remove.
- **Dependencies**: `govulncheck` in CI, `npm audit --omit=dev`, Renovate weekly PRs.
- **Account abuse**: hCaptcha on signup + anonymous `/check` after first miss, exponential backoff on failed login, IP-based block at Caddy for repeated 401s.
- **Secrets in repo**: only `.env.example`. Prod env loaded from `/etc/alethea/env` (root-only, mounted read-only into container).
- **Headers**: CSP that disallows inline scripts in the web app, Referrer-Policy `no-referrer`, Permissions-Policy locked down.
- **2FA (TOTP)**: optional for free, **required for admin and journalist accounts**.
- **`security.txt`** at `/.well-known/security.txt` with disclosure email.
- **No PII in logs**: structured logging with a redactor for emails/keys.

### Why no Postgres RLS

RLS is only useful when *multiple* clients hit Postgres directly (Supabase pattern). Since only the Go API talks to the DB, auth is enforced in Go and RLS is not needed. Easier to reason about, harder to misconfigure.

---

## 5. Phased roadmap

Each phase has a clear "done" condition.

### Phase 0 — Restructure & dev infra ✅ in progress

- Move `factcheckerweb/{backend,frontend,instagram-service}` → `apps/api`, `apps/web`, `services/scrapers/instagram`
- Rename Go module to `alethea/api`
- Promote `.git` to repo root (preserves history)
- Top-level: README, LICENSE, .gitignore, .env.example, Makefile
- `infra/docker-compose.dev.yml` with postgres + redis + api + web + scraper
- Per-app Dockerfiles (dev + prod targets where relevant)
- **Done when:** `make dev` boots all services and existing fact-check pipeline still works end-to-end.

### Phase 1 — DB + Auth + RateLimit ✅

- `db/` with goose migrations (embedded) and pgx pool — sqlc deferred until Phase 2+ query growth justifies it
- Tables shipped: `users`, `sessions`, `email_tokens`, `audit_log`
- `/auth/signup`, `/auth/login`, `/auth/logout`, `/auth/me`, `/auth/forgot`, `/auth/reset`, `/auth/verify`
- Argon2id password hashing (OWASP 2025 params), opaque session cookies (HttpOnly/Secure/SameSite=Lax), CSRF double-submit middleware tied to a per-session token, rate-limit middleware (Redis sliding window)
- Account lockout after N failed logins (default 8 → 15min lock)
- Audit log appended for every auth-significant event
- Structured logging (`log/slog`) with PII redactor (emails, bearer tokens, sensitive keys)
- hCaptcha integration (NoCaptcha fallback when secret not set — dev only)
- LogMailer for dev email links; SMTPMailer wired in Phase 7
- Env-driven CORS (`ALLOWED_ORIGINS`), config struct centralized
- Unit tests (password, tokens) + integration test scaffolding (gated on `TEST_DATABASE_URL`)

### Phase 1.5 — Social login (before mobile)

- OAuth: Google, Apple (mandatory if any social login on iOS), magic-link
- 2FA TOTP enrolment flow (DB columns already exist)
- Should land before Phase 6 mobile work begins.

### Phase 2 — Pipeline v2 ✅

Implemented as **SRIR-Hybrid** (Shared Retrieval + Independent Reasoning + screener question-fanout + graceful BYOK fallback) — see the design workflow output. Packages shipped:

- `services/factcheck/` — orchestrator, screener, retrieval, investigator (3 stylistic roles), judge with skeptical floor + intent + steelman, content_hash (non-repudiation), canonicalize, highstakes regex
- `services/factcheck/cache/` — Redis verdict cache (key = prompt_version|judge_model|claim_hash, TTL per verdict kind)
- `services/factcheck/checkstream/` — SSE hub with ring buffer + replay-from-seq + 10-min post-completion hold
- `services/factcheck/persist/` — DB CRUD for the new tables
- `services/search/` — Provider interface + Brave + Tavily + Multi (fallback chain) + Redis per-query cache + MockProvider for tests
- `services/ai/` — Provider interface extended with `CompleteJSON`, `ChatWithSystemCtx`, `ModelID`. Mistral fully implemented; OpenAI/Anthropic/OpenRouter STUB (Phase 2.5 to flesh out)
- `services/ai/aimock/` — deterministic test mock
- `services/byokresolver/` — picks user's BYOK key or pool, graceful fallback
- `byok/` — AES-GCM vault with AAD binding (userID|provider), tested
- `httpx/safefetch.go` — SSRF-guarded URL fetcher; rejects RFC1918/loopback/link-local/metadata/ULA; re-checks IP at connect (TOCTOU)
- `promptsafe/` — `<<<UNTRUSTED-DATA>>>` delimiter wrapper + system boilerplate; defangs forged delimiters
- `handlers/check_v2.go` — pipeline-backed `/api/check`, `/api/check/{id}`, `/api/check/{id}/stream`
- `handlers/byok.go` — `/api/me/keys` GET/POST/DELETE
- Migrations 00006-00009: checks, claims, agent_runs+verdicts+citations, byok_keys

Decisions taken on the open questions:
- Brave primary, Tavily fallback. No DDG (ToS-grey).
- Graceful BYOK fallback with SSE `byok_fallback` event + audit log.
- 3 investigator styles (empiricist/skeptic/historical-context), default at N=1 = empiricist.
- Domain diversity relaxed to ≥2 with confidence cap of 0.55 when only 2 (rather than hard fail).
- Anonymous checks are readable by UUID (122-bit entropy).
- Per-tier daily $ ceiling: not enforced in v1 — operational tuning + alerting once we see real traffic.

**Done when:** N=3 investigators produce a dissent vector in the API response; cache hit serves a repeat claim in <200ms (integration test asserts this). Integration tests in `services/factcheck/pipeline_integration_test.go` cover happy path, cache hit, skeptical-fallback, retrieval failure, BYOK roundtrip.

### Phase 2.5 — Provider rollout + adversarial bench (post-launch)

- Real OpenAI + Anthropic + OpenRouter providers (currently stubs)
- Per-tier daily $ ceiling enforcement
- Adversarial benchmark suite (~100 known-misinformation + ~100 known-true + ~50 edge cases). Score published as `/dashboard` for radical transparency.
- **Deferred Phase 2 security findings** (filed by the multi-lens adversarial review, judged acceptable for now — fix before scale-up):
  - Resolver.For does a synchronous DB read per check; under pool saturation it blocks pipeline start. Mitigation: cache user plan + BYOK presence in the session row.
  - Plaintext BYOK key copy lives in `ai.Config` for the request lifetime — Go strings are immutable, can't truly zeroize. Mitigation: switch the provider config to a `[]byte` field that GC can wipe.
  - Redis cache values are unauthenticated — a compromised Redis can inject arbitrary verdicts. Mitigation: HMAC the cache value with a server secret.
  - Brave + Tavily HTTP clients are plain `http.Client` (no SSRF guard), but they only talk to two fixed public hosts. Acceptable.
  - `unverifiable` TTL is 1h — caches transient search outages. Tune to 5min for v1.5.
  - Concurrent `InsertCitation` for the same `agent_run_id` has no idempotency guarantee — duplicates would inflate counts. Need a `(agent_run_id, url)` unique index.
  - NFKC canonicalization may collapse two semantically distinct claims (e.g. with vs. without superscript) to the same hash. Acceptable false-positive rate for cache; revisit if observed in practice.

### Phase 3 — Sources & Journalist program (3-5 days)

- Seed source DB with ~100 reputable outlets (Reuters, AP, BBC, Le Monde, Der Spiegel, Nature, peer-review aggregators, .gov, .edu)
- Investigators prefer curated sources at retrieval + rerank
- Journalist application form + admin queue + decision flow
- "Vetted source" badge in result UI
- **Done when:** admin approves an application, that journalist's outlet appears as a higher-tier source within a subsequent check.

### Phase 4 — Web redesign (4-6 days)

- Design tokens (dark + light), distinctive palette (suggest deep teal + violet over pure black-blue)
- Framer Motion: verdict ring count-up, agent-vote bar fill, citation cards staggered in, search-progress timeline
- Pages: landing, /check, /history, /settings/keys, /how-it-works, /sources, /journalist-program, /economics, /pricing, /login, /signup
- A11y pass: keyboard, ARIA, contrast AAA in dark mode
- i18n scaffold from day 1 (EN/DE/ES/FR strings)
- **Done when:** Lighthouse a11y ≥ 95; every verdict is fully traceable to citations in ≤3 clicks; both themes work.

### Phase 5 — Billing (3-4 days)

- Stripe Checkout + Customer Portal
- Webhooks → `plan_events` → `users.plan`
- Donations page → Open Collective + GitHub Sponsors
- **Done when:** test card upgrades a user to Plus and rate limit reflects the new plan.

### Phase 6 — Mobile (Expo) (1-2 weeks)

- Expo Router app under `apps/mobile`, sharing `packages/shared-types`
- Native share extension on iOS, share intent on Android: "Share to Alethea" from any post → opens with URL pre-filled
- Auth + BYOK + check + result + history screens
- Optional push notifications on long checks
- **Done when:** TestFlight build runs the full share → check → verdict flow; internal Android track does the same.

### Phase 7 — Deploy (2-3 days)

- Hetzner CX22+ (4GB) for v1, CCX13 (8GB) once Plus is active
- `Caddyfile` reverse-proxies api/web with auto-TLS
- `docker compose up -d` as systemd unit
- Cron: nightly encrypted `pg_dump` → Storage Box
- Uptime Kuma → Telegram alerts
- **Done when:** `alethea.<domain>` serves, TLS A grade, monitoring fires test alert.

### Phase 8 — Launch

- `docs/{how-it-works,trust-policy,source-policy,journalist-program,TOS,privacy,security}.md`
- `/.well-known/security.txt`
- App Store + Play Store submissions
- HN Show, Reddit (r/journalism, r/skeptic, r/selfhosted), Mastodon, journalism mailing lists

### Phase 8.5 — Distribution multipliers (post-launch)

- **Browser extension** (Chrome/Firefox): right-click → fact-check drawer. Highest-leverage habitual-use feature.
- **Discord + Slack bots**: paste a link, get verdict inline.
- **Embed widget**: `<iframe src="alethea.app/embed/{id}">` for journalists to show verified claims on their own sites.

---

## 6. Human-only TODO list

Things only you can do — track separately from this codebase work.

### Pre-build (parallel with Phase 0-1)

- [ ] Domain — check `alethea.app`/`.org`/`.ai` availability + trademark conflicts (Alethea is Greek for truth; consider `getalethea`/`alethea.fact`/`alethea.is`)
- [ ] VPS — Hetzner (€5-15/mo) + SSH key + non-root user + ufw + fail2ban
- [ ] Email provider — Postmark / SES / Mailgun + SPF + DKIM + DMARC
- [ ] LLM provider account(s) — Mistral + OpenAI/Anthropic/OpenRouter for fallback; hard monthly spend caps in dashboards
- [ ] Search API — Tavily or Brave
- [ ] hCaptcha account (or Cloudflare Turnstile — free)

### Legal / policy (before public launch)

- [ ] Privacy Policy + TOS — iubenda/Termly template; **have a lawyer review** if EU users
- [ ] Cookie/consent banner if collecting analytics
- [ ] DPA template for journalist/enterprise users (later)
- [ ] DMCA/abuse contact email — `abuse@alethea...`
- [ ] Responsible disclosure (`/.well-known/security.txt`)
- [ ] Content moderation policy + takedown process

### Money / accounts

- [ ] Stripe account — KYC, Stripe Tax for VAT/EU, Customer Portal config
- [ ] Open Collective fiscal host (or GitHub Sponsors)
- [ ] Apple Developer Program ($99/yr)
- [ ] Google Play Developer ($25 one-time)
- [ ] Business bank account (optional)
- [ ] Legal entity decision (sole prop fine to start; LLC/UG later for liability)

### Store assets

- [ ] App icon (1024×1024) + adaptive Android icon
- [ ] Screenshots: iPhone 6.7"/6.5"/5.5", iPad, Android phone+tablet
- [ ] App Store description, keywords, support URL, privacy URL
- [ ] Demo video

### Ops / on-call

- [ ] Support SLA documented (even "best-effort 72h")
- [ ] Public Discord / GitHub Discussions
- [ ] Status page (UptimeRobot free tier)

### Things to think about

- Pricing — €5/mo is a guess; revisit after 3 months of usage
- Non-profit / EU public-interest grant status (NLnet, Sovereign Tech Fund)
- Translation strategy beyond launch — DE/ES/FR first, then expand

---

## 7. Decisions log

| Date | Decision | Rationale |
| --- | --- | --- |
| 2026-06-22 | Mobile: Expo (React Native) | Code-share with React web; solo-maintainable. Existing Swift app retires after Expo iOS works. |
| 2026-06-22 | Backend: evolve in-place (no rewrite) | Keep working pipeline, prompts, SSE plumbing. |
| 2026-06-22 | Auth: Postgres + Go sessions (no RLS) | Only Go talks to DB; RLS not needed. |
| 2026-06-22 | Go module renamed `fact-checker` → `alethea/api` | Aligns with monorepo path; clear ownership. |
| 2026-06-22 | Verdict taxonomy unchanged (7 categories) | Existing categories cover the needed nuance. |
| 2026-06-22 | Migrations live with the binary at `apps/api/db/migrations/` (go:embed) | Self-contained prod binary; no external mount needed. |
| 2026-06-22 | sqlc deferred until Phase 2+ | Phase 1 queries are simple enough that hand-written pgx is faster to ship. |
| 2026-06-22 | OAuth deferred to Phase 1.5 (before mobile) | Keep Phase 1 shippable; password auth is enough for web launch. |

---

## 8. Open questions

- Whether `/economics` page belongs in Phase 4 (web redesign) or Phase 5 (billing) — leaning Phase 5 since it depends on Stripe MRR.
- Verdict cache invalidation strategy for "evergreen" claims that drift over time — likely a re-check schedule per Plus subscription.
- App name conflict check still pending — may need to qualify Alethea with a suffix.
- Whether to ship a public read-only verdict gallery (opt-in by checker) — boosts discovery, adds moderation burden.
