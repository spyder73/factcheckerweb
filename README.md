# Alethea

**Open-source, evidence-based fact-checking.** Paste any social-media
post or claim and get a verdict you can trace: every claim, every
source, every agent vote — fully visible. Multi-agent under the hood,
skeptical by default, transparent about its own uncertainty.

> Alethea (ἀλήθεια) — Greek for *truth*, also the *absence of forgetting*.

---

## What it does

1. You paste a URL or some text.
2. We scrape it, analyze any images, extract atomic claims.
3. A pool of independent investigator agents researches each claim
   against a curated source registry + live web search.
4. A judge model synthesizes their reports into a final verdict with
   citations, dissent, and a confidence score.
5. You see the whole thing — sources, agent transcripts, the lot — and
   can disagree on the evidence rather than on faith.

See [`docs/how-it-works.md`](docs/how-it-works.md) for the long version.

## Why it's different

- **Skeptical default.** When the evidence is thin, the verdict is
  "not enough evidence," never a confident guess. A hard confidence
  floor at 0.65 enforces this.
- **Multi-agent.** N independent investigators see the same source
  pool but reason separately; the judge has to acknowledge dissent
  explicitly.
- **Open everything.** Pipeline code, prompts, source list, and trust
  tiers are all in this repo. Self-hostable.
- **BYOK supported.** Bring your own Anthropic / OpenAI / Mistral /
  OpenRouter key and route only through that provider.
- **Refuses to fact-check the wrong things.** Opinions, predictions,
  personal experiences, and humor get a labeled refusal instead of a
  fabricated verdict.

## Status

Pre-1.0. Phases 0–8 of the build plan are complete on `main`; what
remains is human work (Stripe live keys, App Store registration, legal
review of `docs/TOS.md` + `docs/privacy.md`, production DNS). See
[`docs/PLAN.md`](docs/PLAN.md) for the trajectory and
[`docs/CHANGELOG.md`](docs/CHANGELOG.md) for what landed when.

## Quickstart (development)

Requires Docker Desktop + a [Mistral][mistral] API key (or set
`AI_PROVIDER=openrouter` and use an [OpenRouter][openrouter] key).

```sh
cp .env.example .env             # set MISTRAL_API_KEY (or OPENROUTER_API_KEY)
make dev                         # docker compose up — postgres, redis, api, web, scraper
```

Open <http://localhost:3000>. API at <http://localhost:8080>.

Run targets individually:

```sh
make api       # go run apps/api (needs an LLM key in .env)
make web       # vite dev server
make scraper   # python services/scrapers/instagram
make down      # stop everything
make logs      # tail compose logs
```

## Project layout

| Path                           | What |
| ------------------------------ | ---- |
| `apps/api/`                    | Go backend — chi router, multi-agent pipeline, Postgres, Redis, BYOK vault, Stripe billing |
| `apps/web/`                    | React + Vite + TS + Tailwind frontend |
| `apps/mobile/`                 | Expo Router (iOS + Android) — scaffolded |
| `packages/shared-types/`       | TS types canonical across web + mobile |
| `services/scrapers/instagram/` | Python service wrapping Instaloader |
| `infra/`                       | docker-compose (dev + prod), Caddyfile, systemd units, backup script |
| `docs/`                        | PLAN, DEPLOY, how-it-works, trust + source policies, TOS, privacy, security, contributing, changelog |

## Deploying it yourself

Read [`docs/DEPLOY.md`](docs/DEPLOY.md). It walks from a cold Debian
VPS to live HTTPS, including Stripe webhook wiring, automated
encrypted backups, and a quarterly disaster-recovery drill.

## Contributing

[`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md). Source-list improvements
and prompt-quality patches are especially welcome.

Security disclosure: `security@alethea.example` —
[`docs/security.md`](docs/security.md).

## License

MIT. See [`LICENSE`](LICENSE). The curated source list and verdict
templates are dual-licensed CC-BY-SA for editorial reuse.

---

Built by a small team and a lot of independent agents. If a verdict on
the live site looks wrong, file an issue with the verdict ID — we read
every report.

[mistral]: https://console.mistral.ai/
[openrouter]: https://openrouter.ai/
