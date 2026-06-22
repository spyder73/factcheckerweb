# Alethea

**Open-source, evidence-based fact-checking for everyone.**

Paste any social-media post (or claim) and get a traceable verdict: every claim, every source, every agent vote — fully visible. Multi-agent under the hood, skeptical by default, transparent about its own uncertainty.

Status: **early development**. The current codebase is a single-agent PoC being evolved into the full product. See [`docs/PLAN.md`](docs/PLAN.md) for the build roadmap.

---

## Monorepo layout

| Path | What |
| --- | --- |
| `apps/api/` | Go backend (chi router, SSE streaming, Mistral provider) |
| `apps/web/` | React + Vite + TS + Tailwind frontend |
| `apps/mobile/` | _(Phase 6)_ Expo iOS + Android app |
| `services/scrapers/instagram/` | Python Flask service wrapping Instaloader |
| `packages/shared-types/` | _(Phase 1+)_ TS types generated from Go structs |
| `infra/` | docker-compose, Caddy config, DB migrations, backup scripts |
| `docs/` | Plan, methodology, source policy, TOS, privacy, security |

---

## Quickstart (dev)

Prereqs: Docker Desktop, Go 1.21+ (only if running the API outside Docker), Node 20+ (only for web outside Docker), Python 3.12+ (only for scraper outside Docker), a Mistral API key.

```bash
cp .env.example .env          # then set MISTRAL_API_KEY
make dev                       # docker compose up — postgres, redis, api, web, scraper
```

Open <http://localhost:3000>. API at <http://localhost:8080>. Scraper at <http://localhost:5001>.

Individual targets:

```bash
make api        # go run apps/api (needs MISTRAL_API_KEY)
make web        # vite dev server (apps/web)
make scraper    # python services/scrapers/instagram/app.py
make down       # stop compose
make logs       # tail compose logs
```

---

## Methodology (what Alethea does)

1. **Scrape** the linked post (Instagram via the Python service; Twitter/X, TikTok, YouTube, Facebook via the generic meta-tag scraper; arbitrary HTTPS allowed via SSRF-guarded fetch).
2. **Image analysis** — every attached image goes through a vision model in parallel.
3. **Claim extraction** — a cheap model condenses everything into atomic checkable claims.
4. **Fact evaluation** — _(Phase 2)_ N investigator agents independently research each claim with web search; a strong judge synthesizes a verdict with dissent metrics.
5. **Skeptical fallback** — if confidence is low, the verdict is `Unverifiable`, not a guess.
6. **Intent panel** — what the post is trying to do (sell / persuade / mock / inform), clearly labeled as interpretation.

Every verdict shows its full agent transcript, every cited source, and a confidence score. See [`docs/how-it-works.md`](docs/how-it-works.md) _(Phase 8)_ for the detailed explanation.

---

## License

MIT. See [`LICENSE`](LICENSE).

---

_Alethea (ἀλήθεια) — Greek for truth, also the absence of forgetting._
