# Contributing to Alethea

Thanks for thinking about it. The project is open source so that anyone
can verify how we produce verdicts; we're also happy to take patches,
new sources, prompt improvements, and translations.

## Quick start

```sh
git clone https://github.com/alethea-app/alethea.git
cd alethea
cp .env.example .env       # fill in MISTRAL_API_KEY at minimum
make dev                   # docker compose up
```

If the API starts and `curl localhost:8080/health` returns
`{"status":"healthy"}`, you're set.

## What we want patches for

In rough order of welcomeness:

1. **Source list improvements.** Add domains with proper tier
   justification, fix misclassifications, document a beat-specific
   override. PRs go to
   `apps/api/db/migrations/0001N_sources.sql` with a one-paragraph
   rationale per change.
2. **Prompt improvements.** The screener / investigator / judge
   prompts live in `apps/api/services/factcheck/*.go`. If you can
   show — with a small replay — that a prompt change improves
   verdict quality on a tough class of inputs, we want the PR.
3. **Translations.** EN is canonical. DE is translated. ES + FR are
   stubs. PRs adding/fixing strings in `apps/web/src/i18n/*.json`
   are welcome.
4. **Bug fixes.** Reproduce instructions + a failing test help a lot.
5. **Pipeline hardening** from `docs/PIPELINE_HARDENING_TODO.md`.
6. **Mobile app** completion. The scaffolding is in `apps/mobile/`
   with a TODO list at the top of `apps/mobile/README.md`.
7. **Browser extension / Slack bot / Discord bot.** Phase 8.5; we'd
   review and merge a working PR for any of them.

## What we will not accept

- "AI-generated PRs" without you understanding what they do. We will
  ask follow-up questions; if you can't answer, we'll close.
- "Adds my SaaS as a feature." We are not a marketing channel.
- Large refactors with no functional change. They make code review
  hard; if you think something needs restructuring, open an issue
  first and we'll discuss.
- New dependencies for trivial features. We're tight on the supply
  chain because the project's whole credibility depends on the
  pipeline being inspectable; every new dep is more surface.
- Changes to the verdict copy or trust-tier definitions without
  discussion. Open an issue; we'll talk.

## Code style

- **Go**: standard library where it suffices. `gofmt -s -d`,
  `go vet`, and `golangci-lint` should pass (the latter when CI is
  wired). Prefer `slog` to `log`. Avoid panics outside `main()`.
- **TypeScript (web)**: strict mode on. No `any` without a comment
  explaining why. Tailwind classes; no random one-off CSS.
- **TypeScript (mobile)**: same as web, plus: avoid platform-specific
  code paths unless we genuinely can't write a cross-platform version.
- **Comments**: explain why, not what. Don't write "// increments x"
  above `x++`. DO write a comment when the code is doing something
  surprising on purpose.

## Tests

- Go: `go test ./...` should pass. New features need at least a
  smoke test; new pipeline stages need a deterministic test using the
  `aimock` provider.
- Web: `npx tsc --noEmit` passes. We don't have a comprehensive unit
  suite (yet) — Playwright is on the roadmap.
- Mobile: `cd apps/mobile && npm run typecheck` passes.

## Commit messages

We don't enforce a format, but:
- Short summary line, ideally < 70 chars.
- Body explains WHY the change is needed. The diff already shows what.
- Reference issues / Phase A item IDs when relevant.
- Don't squash to one commit unless that's actually how the work
  happened.

## DCO / sign-offs

Not currently required, but if you'd prefer to sign off your commits
with `git commit -s`, that's welcome.

## Branching

- `main` is the deploy branch.
- Open PRs against `main`. Long-lived feature branches are fine if
  the work is genuinely incremental.

## Reviewing

A maintainer reviews within ~7 days. We're a small team, so:

- Smaller PRs get merged faster.
- PRs with passing CI + a clear description get merged faster still.
- We will sometimes ask you to split a large PR. Sorry — it makes
  the review tractable.

## Security issues

DO NOT open a public PR or issue. Email `security@alethea.example`.
See [`security.md`](security.md) for the disclosure policy.

## Community

- GitHub Discussions for questions, ideas, RFCs.
- Mastodon `@alethea@TBD` for project updates (announced before launch).
- We don't have a Slack/Discord yet. If demand justifies it, we'll
  create one.

Thanks again. Even small PRs help — fixing a typo in the verdict copy
is a real contribution.
