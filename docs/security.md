# Security policy

**The contact + disclosure policy for security researchers.** What
Alethea promises about security practice lives in
[`trust-policy.md`](trust-policy.md) and the source code. This document
is for telling us something is broken.

## Reporting a vulnerability

Email `security@alethea.example`. Encrypt with our PGP key if the
vulnerability is sensitive (see "Key" below).

Please include:

- A short description of the issue.
- Steps to reproduce, ideally with a minimal proof-of-concept.
- The affected version (git commit SHA if possible).
- Whether you've already disclosed it elsewhere.

**Response SLA**: we acknowledge receipt within 72 hours and aim to
have a fix in the main branch within 14 days for high-severity issues,
30 days for medium, 90 days for low. If we miss the SLA we'll tell
you why.

## In scope

Anything in this repository, including:

- The Alethea web app (`apps/web/`).
- The Alethea API (`apps/api/`).
- The mobile app (`apps/mobile/`) once it's published.
- The deployed instance at `alethea.example` (or whatever domain we
  end up on — check the canonical URL in `/.well-known/security.txt`).
- Our Stripe webhook handling.
- Our BYOK vault.

Specifically welcome:

- **Auth or session bypass.** Forging a session cookie, escalating
  privileges, accessing another user's data via predictable IDs.
- **CSRF**, despite our double-submit defense.
- **XSS / template injection** in the web app or any API response.
- **SSRF** via the URL submission flow or any scraper. The DNS-resolve-
  then-reconnect guard is in `apps/api/httpx/ssrf.go`; if you find a
  bypass, that's a CVE-level issue.
- **SQL injection.** We use parameterized queries everywhere, but if
  you find a string-concat slip, please report.
- **BYOK extraction.** If you can decrypt another user's BYOK key,
  that's a critical issue. The AEAD AAD includes (user_id, provider)
  so a stolen ciphertext shouldn't be replayable; if you find a
  break, report.
- **Stripe webhook bypass / signature forgery.** Our webhook validates
  the Stripe signature header and uses `plan_events.stripe_event_id`
  UNIQUE for idempotency.
- **Cross-tenant data leakage** on any multi-user endpoint.
- **Cache-key collisions** that let one user read another's verdict.
- **Rate-limit bypass** that costs us money (most rate-limit bypasses
  don't, so judge accordingly).
- **Pipeline prompt injection** that lets a submitted URL break out of
  the `<<<UNTRUSTED-DATA>>>` delimiters and execute the system prompt.

## Out of scope

- Volumetric DoS. We acknowledge it's possible to overwhelm us with
  traffic; please don't demonstrate.
- Vulnerabilities in third-party services (Stripe, Anthropic, Mistral,
  Brave). Report those to the respective companies.
- Issues that require physical access to a user's device.
- Social engineering against us or our users.
- Reports from automated scanners with no proof of exploitability.
  We will not respond to "you have port 443 open" tickets.
- Missing security headers on subdomains we don't actually own.
- Self-XSS that requires the victim to paste attacker-controlled JS
  into their own console.

## Coordinated disclosure

Default: 90 days from acknowledgment, OR the day we ship a fix,
whichever is earlier. We'll discuss extensions for issues that need
upstream coordination (e.g. a flaw in a dependency).

Once a fix is shipped, we'll:

- Credit you in the release notes (unless you prefer to stay anonymous).
- Issue a CVE if applicable.
- Write a post-mortem in `docs/security-advisories/` (created when the
  first one lands).

## Hall of fame

The list of researchers who have responsibly disclosed issues. To be
populated. We will not pay bounties in v1 (no budget), but every
researcher will be credited unless they prefer anonymity, and we'll
write you a public thank-you on Mastodon / Bluesky / HN if you want.

## Key

PGP key for sensitive reports: TBD before launch. Until the key is
published, please send unencrypted email with a note saying you have a
sensitive report; we'll respond from a key we then publish.

Fingerprint of `security@alethea.example` will be cross-posted to:
- This document.
- `/.well-known/security.txt`.
- The maintainer's HN / Mastodon profile.

## Internal security commitments

Things we promise to do, beyond responding to reports:

- Keep dependencies current. Dependabot is wired in the repo.
- Run `go vet ./...` + the test suite in CI on every PR.
- Regenerate the BYOK master key if we suspect compromise (will
  invalidate every existing BYOK key — users will need to re-enter).
- Verify backups restore correctly at least quarterly.
- Maintain an audit log of admin actions in `audit_log`.
- Never log BYOK ciphertexts or session tokens.
- Use only minimal scopes when talking to third-party APIs.

Things we will eventually do but haven't yet:

- Public bug bounty program (needs budget).
- Independent security audit (needs budget).
- SOC2 / ISO 27001 (needs sufficient enterprise demand to justify).
- HSM-backed BYOK master key (currently a base64 string in env).
