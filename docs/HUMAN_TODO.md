# Human-only TODO

Everything that requires a real person — not code. Roughly grouped from
"must do before you can launch" down to "do later." Items are deliberate
about what's blocking what.

Quick sanity:
- ☐ = not done
- 🚧 = in progress / partial
- ✅ = done (move it down to bottom when so)

Items that are gated on money are flagged 💶.

---

## 1. Accounts to create (before you can launch)

- ☐ **Domain.** Pick + buy. Suggestions: `alethea.app`, `alethea.fyi`, `alethea.news`. Avoid `.io` (price) and ccTLDs that need local presence. ~€15/yr.
- ☐ **GitHub organization.** `alethea-app` or similar. Move this repo into it; transfer ownership of the repo URL referenced throughout the codebase.
- ☐ **Cloudflare account** for the domain (free tier is fine). DNS + DDoS shield.
- ☐ **Hetzner Cloud account** for the VPS. ~€5/mo for CX22, €15/mo for CCX13 once Plus is live.
- ☐ **Hetzner Storage Box** for encrypted backups (BX11 €4/mo, 1TB).
- ☐ **Stripe account** (Atlas if you don't have an entity yet — €500 one-time, gets you a US LLC; or set up an EU entity locally).
- ☐ **Resend / Postmark account** for transactional email. Resend's free tier is 3k emails/mo — fine for v1. Wire it to a `noreply@your-domain` address with SPF + DKIM + DMARC.
- ☐ **Brave Search API key.** $3/mo for the dev tier — fine to start.
- ☐ **Tavily API key.** Free tier 1000 searches/mo.
- ☐ **OpenRouter account** + initial credit. Or direct Anthropic / OpenAI / Mistral accounts if you prefer.
- ☐ **Apple Developer Program** membership. 💶 $99/yr.
- ☐ **Google Play Developer account.** 💶 $25 one-time.
- ☐ **EAS (Expo) account** for mobile builds. Free tier works for low volume; €19/mo if you go through their build queue regularly.
- ☐ **Mastodon account** for the project (e.g. on `infosec.exchange` or `fosstodon.org`). Free.
- ☐ **Bluesky account** for the project. Free.
- ☐ **Optional: Open Collective** for transparent donations.
- ☐ **Optional: GitHub Sponsors** — needs a bank account in a supported country.
- ☐ **Optional: Ko-fi / Buy Me a Coffee** for one-off donations.

## 2. Legal (before money / users)

- ☐ **Pick an operating entity.** Solo trader is simplest but exposes you personally; an LLC/UG/Ltd protects you. Get accounting advice.
- ☐ **Find a lawyer** in your jurisdiction familiar with consumer SaaS + GDPR. Ideally also AI/LLM-aware.
- ☐ **Lawyer-review `docs/TOS.md`.** Specifically the bits flagged `[OPERATOR NAME + ADDRESS — fill in]`, the limitation-of-liability cap, and the consumer-revocation language.
- ☐ **Lawyer-review `docs/privacy.md`.** Specifically: data controller info, subprocessor list completeness, the §15/§17 GDPR right-flow, the data-retention table.
- ☐ **Imprint page (`Impressum`).** Required by German law (TMG §5) if you're operating from DE: full legal name, address, contact, USt-ID if registered. Add to web app footer.
- ☐ **Sign DPA (Data Processing Agreement) with each subprocessor.** Stripe and Anthropic provide DPAs out of the box; OpenRouter / Mistral / OpenAI / Brave / Tavily — check each.
- ☐ **Trademark check** on "Alethea" in your operating regions. Common name, common in pharma — might already be taken in your trademark class.
- ☐ **Decide on company structure for donations** if you go that route (e.g. a separate non-profit shell so donations are tax-deductible). Lawyer + accountant.

## 3. Money + billing

- ☐ **Create a Stripe Product** "Alethea Plus" with two Prices:
  - Monthly recurring €10.00 → `price_...` → set as `STRIPE_PRICE_PLUS_MONTHLY`.
  - Yearly recurring €100.00 → `price_...` → set as `STRIPE_PRICE_PLUS_YEARLY`.
- ☐ **Create a Stripe Webhook endpoint** at `https://YOUR-DOMAIN/api/billing/webhook` listening to:
  - `checkout.session.completed`
  - `customer.subscription.updated`
  - `customer.subscription.deleted`
  - `invoice.payment_failed`
- ☐ Copy the webhook signing secret → `STRIPE_WEBHOOK_SECRET` in `/etc/alethea/.env`.
- ☐ **Test with a real card** before launch. Use `4242 4242 4242 4242` for test mode; for live, use your own card with a refund.
- ☐ **Enable Stripe Tax** so EU VAT is collected correctly for B2C subscriptions. Otherwise your invoices are non-compliant in DE/FR/IT/etc.
- ☐ **Decide donation method.** Set `DONATE_URL` env var or hide the donate button (it's hidden by default).
- ☐ **Set up bookkeeping.** Stripe → your accountant's accounting tool (Lexoffice / sevDesk in DE; QuickBooks / Xero elsewhere). Stripe exports as CSV.

## 4. Operations setup (before launch traffic)

- ☐ **Buy + boot the VPS.** Hetzner CX22, Falkenstein (DE) for EU GDPR alignment.
- ☐ **DNS records:**
  - `A` and `AAAA` for `YOUR-DOMAIN` → VPS IPv4/v6.
  - `CNAME` `www` → apex.
  - `MX` if you want incoming email at the domain (Fastmail / Migadu are fine — ~€3/mo).
  - SPF + DKIM + DMARC for Resend / your mail provider.
- ☐ **Follow `docs/DEPLOY.md` end-to-end.** Including the disaster-recovery dry-run at the bottom.
- ☐ **Generate the BYOK master key offline** (`openssl rand -base64 32`) and store the original in 1Password / a hardware key. NEVER commit it. NEVER paste it into a chat.
- ☐ **Generate the age backup key offline** (`age-keygen`). PRIVATE part to 1Password; PUBLIC part to `AGE_RECIPIENTS` env. Without the private key, backups are unrecoverable.
- ☐ **Test the backup restore.** Don't skip this. Untested backups don't exist.
- ☐ **Set up monitoring.** Either:
  - Off-box: Better Stack / Pingdom / UptimeRobot pointed at `/health`. Free tiers exist.
  - On-box: `docker compose -f infra/monitoring/uptime-kuma.yml up -d`.
- ☐ **Wire a notification channel.** Telegram bot DM is lowest friction. Test it fires.
- ☐ **Schedule the backup verify drill.** Calendar reminder, quarterly. If you do this once and forget, the backups will quietly stop working.

## 5. Pipeline + content prep

- ☐ **Hand-curate the seed source list.** `apps/api/db/migrations/00012_seed_sources.sql` has placeholders — review each, expand to ~80–120 domains covering EN + DE + ES + FR where possible.
- ☐ **Pick the production AI provider.** Recommendation: `AI_PROVIDER=openrouter`, default model `anthropic/claude-4.6-sonnet`. Reasons: rate limits are higher than direct Anthropic, you can fail over by changing one env var.
- ☐ **Run the eval harness** (TODO — see `docs/PIPELINE_HARDENING_TODO.md` M2) on 25–50 seed claims to baseline accuracy before any users hit the system.
- ☐ **Translate stub locales.** `apps/web/src/i18n/{es,fr}.json` are stubs; either get them translated or hide the language picker entries until you do.
- ☐ **Get screenshots for the App Store / Play Store / web.** Real verdicts, not mockups. Phone screenshots need iPhone 15 Pro Max + 6.7" Android frames per Apple/Google rules.

## 6. Mobile app prep (Phase 6 follow-on)

- ☐ **Run `eas init`** in `apps/mobile/` to register the project. Paste the resulting EAS project ID into `app.json` → `extra.eas.projectId`.
- ☐ **Generate icons + splash** (1024×1024 PNG). Drop at `apps/mobile/assets/`.
- ☐ **Build the API auth-token path.** The mobile app expects `/auth/login` to return `{sessionToken, csrfToken}` and accept `X-Session-Token`. The web SPA continues to use HttpOnly cookies. See `apps/mobile/README.md`.
- ☐ **Build `/api/me/checks`** endpoint for history list. (Currently returns 404.)
- ☐ **Test the share extension on real iOS hardware.** The simulator doesn't reliably show the app in the share sheet.
- ☐ **App Store Connect:** create the app record, fill metadata (name, subtitle, keywords, support URL, privacy URL, age rating). Submit a TestFlight build.
- ☐ **Google Play Console:** create the app, fill metadata, submit an internal-track build.
- ☐ **Privacy nutrition labels** on both stores. Both apps collect: email, "user content," "diagnostics" — get the matrix right or you'll get rejected.

## 7. Launch sequence (when everything above is ☑)

- ☐ Soft launch to ~20 friends/journalists for a week. Real submissions, real feedback.
- ☐ Fix the top 5 issues from the soft launch.
- ☐ **Show HN.** Post timing matters; aim for Tue–Thu, 9am Pacific.
- ☐ **r/journalism** and **r/skeptic** announcements with a fresh verdict on a topical claim as the lead.
- ☐ **r/selfhosted** announcement focused on the BYOK + self-host story.
- ☐ **Lobsters** post (if you have an invite).
- ☐ **Mastodon thread** from the project account; ping `@bellingcat` and a few large fact-checkers.
- ☐ **DM journalists** you'd like to use it. Personal outreach beats every mass announcement.
- ☐ **Newsletter post** if you have one; talk to one or two journalism newsletters (Press Gazette, Nieman, JournalismAI) about a guest piece.
- ☐ Have the **/economics page** showing real numbers on day one. It's the strongest credibility signal.

## 8. Operational discipline (forever)

- ☐ **Read the audit log** weekly. Suspicious patterns surface here first.
- ☐ **Watch the cost ticker.** If `/api/economics` shows spend > revenue + donations for two consecutive weeks, tighten budget caps or raise prices.
- ☐ **Issue corrections publicly.** A correction is a credibility-builder, not a credibility-killer. Use the `superseded_by` chain.
- ☐ **Update dependencies monthly.** Dependabot opens PRs; review + merge.
- ☐ **Quarterly: DR dry-run** (restore a backup to a throwaway VPS).
- ☐ **Annually: security review.** Walk the whole codebase looking for things that grew complexity since the last review.

## 9. Things you SHOULDN'T do

(Mistakes other founder-operators make.)

- 🚫 Don't ship the live site without lawyer-reviewed TOS + privacy.
  GDPR fines start at 4% of revenue. Don't.
- 🚫 Don't store the BYOK master key in 1Password "Notes" without
  emergency-access. If you get hit by a bus, no one can recover the
  service.
- 🚫 Don't use Gmail for transactional emails. They classify "you
  signed up for X" as spam astonishingly often. Resend / Postmark.
- 🚫 Don't take VC money before you understand whether the unit
  economics work. Fact-checking has low willingness-to-pay; that's not
  a problem you fix with growth.
- 🚫 Don't quietly fix a wrong verdict. The whole project's
  credibility depends on visible corrections. Use the chain.
- 🚫 Don't add features the user isn't asking for. The Phase A review
  flagged 30 items as "v2 or later" — keep them there until v1 has
  signal that the user actually wants them.
- 🚫 Don't add tracking analytics. The privacy policy says we don't;
  keep it that way.
- 🚫 Don't auto-renew without explicit consumer consent. EU consumer
  law disallows dark patterns; Stripe Customer Portal already handles
  this cleanly.

## 10. Things only you can decide

Open questions where the code is silent because they're policy, not
engineering.

- ☐ Will you allow **anonymous verdicts to be indexed by search engines**? Current default: noindex on all user verdict pages (M11 from Phase A). Probably right; reconsider once corrections workflow is mature.
- ☐ Do you want a **public verdict gallery** ("today on Alethea")? Risks: amplifies whatever's posted. Rewards: marketing + transparency.
- ☐ Do you want **community-submitted source proposals** in v1, or wait for v2? Lower-friction onboarding for journalists vs. risk of low-quality additions.
- ☐ What's your **policy on Russian / Chinese state-media sources**? They're authoritative on some topics, propaganda on others. Per-claim overrides aren't implemented yet (see `source-policy.md`).
- ☐ **Refund policy for Plus.** Code currently allows EU 14-day revocation; do you want to extend that to a no-questions 30-day refund as a goodwill policy?
- ☐ **NSFW / sensitive content.** Pipeline doesn't currently differentiate. Should claims about sexual assault, hate speech, etc. get a content warning?
- ☐ **Languages to support at launch.** EN + DE are translated; ES + FR are stubs. Do you ship with stubs, hide them, or wait?
- ☐ **Corrections policy.** Who decides when a verdict gets retracted? In v1 you. Eventually an editorial board (see `trust-policy.md`).

---

## Notes from the build process

Things I (Claude) noticed during the build that might affect priority:

- **Independence review (Phase A) is gold.** Re-read it before you
  ship. The 24 MUST-FIX items are categorized in
  `docs/PHASE_A_FINDINGS.md`. Most that ship-block are now done (M1,
  M4, M8, M10 schema, M11, M12, M13, M17, M20, M23). M2 (eval
  harness), M16 (two-pass extraction), M22 (NLI citation grounding)
  are the meaningful gaps.
- **Cost discipline is everything.** The pipeline runs ~5–8
  investigator calls + a judge call per claim. At Claude 4.6 Sonnet
  rates that's $0.05–$0.15 per check. The $200/day platform cap kicks
  in around 1500–4000 checks. You'll hit it on launch day.
- **Mistral free tier is unusable in prod.** The 1 req/sec limit
  cascades through the pipeline catastrophically. Switch to
  OpenRouter or a paid provider before any real traffic.
- **The mobile app is scaffolding, not shippable.** The backend
  token-auth path is missing; without it, login doesn't work. See
  `apps/mobile/README.md`.

Good luck. The hardest part of fact-checking is psychological — you
WILL be wrong sometimes; the question is whether you own it.
