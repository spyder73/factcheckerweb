# Privacy policy

**DRAFT — LEGAL REVIEW REQUIRED BEFORE PUBLIC LAUNCH.** As with
[`TOS.md`](TOS.md), this captures intent in plain language. Counsel must
review for GDPR compliance before going live.

Last updated: see git history.

---

## TL;DR

- We collect: email, password hash, the things you check, your BYOK
  keys (encrypted), basic session info (IP, user agent, timestamps).
- We don't sell or share your data with advertisers. Ever.
- LLM providers see what you submit. BYOK keeps that scoped to your
  chosen provider only.
- You can export everything we have on you and delete your account
  permanently from Settings → Privacy.

---

## 1. Who we are

The data controller is **[OPERATOR NAME + ADDRESS — fill in]**. Contact
for privacy questions: `privacy@alethea.example`.

If you're in the EU and unhappy with how we've handled your data, you
have the right to lodge a complaint with your national supervisory
authority. For users in Germany: BfDI (https://www.bfdi.bund.de).

## 2. What we collect

### When you visit the site (signed out)

- IP address (used for rate limiting; truncated to /24 for analytics).
- User agent.
- The text/URL you submit + the verdict produced. Anonymous checks are
  auto-purged 24 hours after creation.
- A "client IP" header from our reverse proxy if you're behind one.

We do NOT use any third-party analytics scripts (no Google Analytics,
no Plausible, no Fathom). We do log requests on the server side for
operational purposes (rate limiting, abuse, debugging) — those logs
contain IP + path + status + timing and rotate after 14 days.

### When you sign up

- Email address.
- Argon2id password hash (the plaintext password never reaches our
  database).
- Account creation timestamp.

### When you sign in

- A session cookie (HttpOnly + SameSite=Lax + Secure in production).
- A CSRF token cookie (readable by JS, for double-submit verification).
- Session metadata: IP, user agent, created/last-seen/expires
  timestamps.

### When you submit a check

- Same as anonymous, but linked to your account.
- We store the full submitted URL/text, the verdict, the per-agent
  reasoning, the citations, and a hash of the inputs (for the content
  hash described in `trust-policy.md`).
- Until you delete the check or the account, this is retained
  indefinitely.

### When you add a BYOK key

- The provider name (e.g. "anthropic").
- The key itself, encrypted with AES-GCM. The encryption key never
  leaves our server (we use a master key in env config; you cannot
  decrypt your own key after submission — we just use it on your
  behalf).
- The encrypted ciphertext is bound by AAD to your user ID + provider
  string, so a stolen ciphertext can't be replayed for a different
  user or provider.

### When you upgrade to Plus

- A Stripe Customer ID (a `cus_...` string).
- A history of plan-change events tied to that customer.
- We do NOT store credit card numbers, CVCs, or expiry dates. Stripe
  handles all of that (PCI scope SAQ-A).

### When you apply for the journalist program

- The form fields you submitted (name, outlet, byline URLs, bio).
- Your application is reviewed by a maintainer and either approved or
  rejected. The data is retained while your account exists.

## 3. What we do with it

| Data | Why |
| --- | --- |
| Email | Sign-in identity, password reset, billing receipts (if you go Plus), security notifications |
| Password hash | Sign-in |
| Submitted content | Producing the verdict; debugging when it's wrong |
| BYOK keys | Making API calls to your chosen provider on your behalf |
| IP / user agent | Rate limiting, abuse detection, session sanity ("you signed in from a new IP") |
| Stripe customer ID | Billing, refunds, cancellations |
| Audit log entries | Security investigation, GDPR data export, regulator response |

## 4. Who we share it with

- **LLM providers** (Anthropic, OpenAI, Mistral, OpenRouter, Google
  via OpenRouter). They see the content you submit, the screener +
  investigator + judge prompts, and our system prompt. They do NOT
  see your account email, your IP (we proxy through our server), or
  your BYOK key (when you BYOK, the key goes only to the provider
  whose key it is).
- **Search providers** (Brave, Tavily). They see the search queries
  we issue, which are derived from the normalized claim and the
  uncertainty questions. They do not see the original input verbatim.
- **Stripe** for payments. They see your card details (which you
  enter on a Stripe-hosted page, never on ours), your email, and the
  plan you purchased.
- **Our hosting provider** (currently Hetzner). They have physical
  access to the disks; the data on disk is protected by Postgres
  authentication + BYOK ciphertexts + encrypted backups.
- **Authorities** when legally required. We don't pre-comply — we
  require a binding order and we tell the user unless we're prohibited
  from doing so.

We do NOT:

- Sell data to advertisers.
- Share data with data brokers.
- Use your submissions to train AI models. (We may publish anonymized
  aggregates on the /economics page: total spend, total checks,
  BYOK%. Never individual submissions.)

## 5. Where the data lives

Currently: a single Hetzner VPS in **Falkenstein, Germany** (EU). DB
backups go to a Hetzner Storage Box, also in Germany, encrypted with
`age` so the storage operator can't read them either.

If we move servers, we'll update this and notify signed-in users by
email at least 30 days before any cross-jurisdiction move.

## 6. How long we keep it

| Data | Retention |
| --- | --- |
| Anonymous check + verdict | 24 hours, then hard-deleted |
| Signed-in user's check + verdict | Until the user deletes it or their account |
| Session rows | 30 days from last seen |
| Email tokens (verify, password reset) | Single-use, expire in 1h |
| Audit log entries | 7 years (legal retention) — user_id NULL'd on account delete |
| BYOK keys | Until the user removes them or deletes the account |
| Stripe customer | Until the user deletes the account; we log the Stripe customer ID so an operator can manually cancel any active subscription |
| Backups | 30 days encrypted, then deleted |

When you delete your account: every row with a foreign key to your user
id is cascade-deleted, EXCEPT audit log rows, where the `user_id` is
set to NULL (so we keep the "an action happened at time T from IP X"
record for legal/regulator response, but it's no longer linked to you).

## 7. Your rights (GDPR)

You have the right to:

- **Access** — Download every row we have linked to you, as JSON.
  Settings → Privacy → Export my data.
- **Erasure** — Permanently delete your account and everything tied
  to it (subject to audit-log retention above). Settings → Privacy →
  Delete my account. Note: this does NOT cancel any active Stripe
  subscription automatically — open the Customer Portal first if you
  have one.
- **Rectification** — Update your email address. Currently there's
  no UI for this in v1; email `privacy@alethea.example` and we'll do
  it within 30 days.
- **Portability** — The data export is JSON, structured, machine-
  readable. You can take it to another service.
- **Objection / restriction** — Email us and we'll discuss. In
  practice this means closing your account.
- **Withdrawing consent** — Same as erasure.
- **Lodging a complaint** with your national supervisory authority.

We will respond to any exercised right within 30 days (often within
72 hours for self-service actions like export + delete).

## 8. Cookies

We set three cookies:

| Name | Type | Purpose |
| --- | --- | --- |
| `alethea_session` | HttpOnly, Secure, SameSite=Lax, 30 days | Authentication. Required. |
| `alethea_csrf` | Readable by JS, Secure, SameSite=Lax, 30 days | CSRF double-submit token. Required. |
| Theme preference | localStorage (not a cookie, technically) | Remember your dark/light choice. Optional. |

We do NOT use any tracking cookies, ad-network cookies, or analytics
cookies. There's no cookie banner because everything we set is
strictly necessary or chosen by you.

## 9. Children

The service is not aimed at children under 16. We do not knowingly
collect data from anyone we believe to be under 16. If you think a
child has signed up, email `privacy@alethea.example` and we'll
investigate + delete.

## 10. Changes to this policy

We may update this policy. Material changes (e.g. new subprocessor,
changed retention period) are notified by email at least 30 days in
advance for existing users. The current version is always at this URL
and the change history is in our public git repo.

## 11. Contact

- General privacy questions: `privacy@alethea.example`
- Data export / deletion: use Settings → Privacy in the app
- Security disclosures: `security@alethea.example`
