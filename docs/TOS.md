# Terms of Service

**DRAFT — LEGAL REVIEW REQUIRED BEFORE PUBLIC LAUNCH.** This document
captures the project's intent in plain language. Before going live, a
qualified lawyer in the operating jurisdiction (currently planned: EU,
Germany) must review it and convert it into a legally binding document
that complies with local consumer-protection, data-protection, and
e-commerce regulations.

Last updated: see git history.

## Plain-language summary

- Alethea is an open-source fact-checking tool. We try hard to be
  accurate but verdicts are produced by a multi-agent LLM pipeline and
  CAN BE WRONG. You should read the cited sources before relying on a
  verdict.
- You're responsible for what you submit. Don't paste copyrighted
  content you don't have the right to share, don't paste private
  information about other people, don't try to use Alethea to attack
  other systems.
- We're responsible for keeping the service running with reasonable
  effort but make no uptime promises. We may take it offline.
- We don't sell your data. We collect what we need to run the service
  and nothing more. See [`privacy.md`](privacy.md) for details.
- If you pay for Plus, you can cancel anytime and we won't bill you
  again. Refunds within 14 days of purchase are allowed without
  cause (EU consumer law).
- Disputes go to a German court unless you live somewhere with
  stronger consumer protections.

The rest of this document expands on each of these.

---

## 1. What Alethea is

Alethea is a web and mobile application that, given a URL or text
input, returns a fact-check verdict produced by a multi-agent LLM
pipeline. The verdict includes a verdict label, a confidence score,
a list of cited sources, and the per-agent reasoning. Source code is
open and available on GitHub.

The service is operated by **[OPERATOR NAME + ADDRESS — fill in
before launch]** ("we", "us", "the operator").

## 2. Eligibility

You must be at least 16 years old (the GDPR consent age in most EU
member states) or have parental consent. You must not be barred from
using the service under applicable law.

## 3. Verdicts are NOT legal, medical, financial, or scientific advice

Read this twice.

- Alethea verdicts are not legal advice, not medical advice, not
  financial advice, not scientific advice.
- Verdicts can be wrong, biased, out of date, or based on retracted
  sources. The cited sources may themselves be wrong.
- You are responsible for verifying any verdict against the cited
  sources before acting on it. If you would not act on the basis of a
  Wikipedia article alone, do not act on the basis of an Alethea
  verdict alone.
- The "WELL-SUPPORTED" label means "multiple independent reputable
  sources confirm this," not "this is the truth."

## 4. Your content

When you submit a URL or text:

- You confirm you have the right to submit it (you wrote it, you have
  the right to share it, or it's publicly accessible content).
- You grant us a license to process it through the pipeline (which
  includes sending it to third-party LLM providers — see "Subprocessors"
  below).
- We may cache the input + verdict for performance + audit purposes.
  Anonymous checks are purged after 24 hours; signed-in users' checks
  are retained until they delete them or their account.

## 5. Prohibited use

You may not:

- Submit content you don't have rights to (copyrighted articles you
  haven't paid for, private messages, etc.).
- Submit content that is unlawful in the operating jurisdiction
  (child sexual abuse material, terrorist content, defamation, etc.).
- Use Alethea to attack other systems (no SSRF via crafted URLs, no
  DoS via mass submission, no probing for vulnerabilities without
  prior coordination through `security@alethea.example`).
- Resell our service or rebrand the verdicts as your own product.
  Embedding individual verdicts in journalism is fine; building a
  competitor with our API as the backend is not.
- Train AI models on our verdicts. The verdicts are CC-BY-SA for
  human readers and journalism use; the LLM training carve-out is
  separate and not granted by these terms.
- Circumvent rate limits or budget caps.

## 6. Account suspension

We may suspend or terminate access if you violate Section 5. We will
make a reasonable effort to tell you why and give you an opportunity
to appeal. Material violations (child abuse content, attacks on the
system) are grounds for immediate termination without warning.

## 7. Subprocessors

Your input is processed by third-party LLM providers (the current
list, subject to change): **Anthropic (Claude), OpenAI (GPT),
Mistral, OpenRouter (which routes to the above + Google Gemini),
Brave Search, Tavily**. We do not control how those providers
process the data; we route to them under their own terms.

When you use BYOK, your input goes ONLY to the provider whose key
you supplied. The key itself never leaves our server in plaintext
(AES-GCM encrypted at rest).

If you're processing personal data of EU residents, you need to
satisfy yourself that the subprocessors above meet your GDPR
obligations.

## 8. Pricing + cancellation

- Anonymous + Free + BYOK tiers are free.
- Plus tier is **€10/month** (or **€100/year**) billed via Stripe.
- You can cancel anytime via the Customer Portal. Cancellation takes
  effect at the end of the current billing period.
- EU consumer law: you may revoke your purchase within **14 days**
  of subscribing for a full refund, EXCEPT if you've actively used
  the Plus tier (made checks that consumed pooled compute) in which
  case we may deduct a usage-proportional amount.

## 9. Service availability + changes

- We make no uptime guarantees.
- We may change the service, including pricing, with reasonable
  advance notice (30 days for material changes; immediately for
  security fixes).
- We may discontinue the service entirely with at least 60 days'
  notice and an export of your data.

## 10. Limitation of liability

To the maximum extent permitted by law:

- We are not liable for indirect, incidental, consequential, or
  punitive damages.
- Our total liability for any claim is capped at the amount you
  paid us in the 12 months before the claim (so €120 if you're on
  yearly Plus, €0 if you're on a free tier).
- Nothing in this section limits our liability for gross negligence,
  willful misconduct, or rights that cannot be waived under
  applicable consumer-protection law.

## 11. Governing law + jurisdiction

These terms are governed by the laws of **[OPERATING JURISDICTION —
fill in]**, without regard to conflict-of-laws principles. Disputes
go to the courts of **[CITY, COUNTRY]**, EXCEPT that consumers in
the EU may bring claims in their own member state.

## 12. Changes to these terms

We may update these terms. Material changes are notified by email
(if you have an account) and shown as a banner on the site for at
least 30 days. Continued use after the effective date is acceptance.

## 13. Contact

- General questions: `hello@alethea.example`
- Privacy / GDPR: `privacy@alethea.example`
- Security disclosure: `security@alethea.example`
- Corrections to verdicts: `corrections@alethea.example`
- Press: `press@alethea.example`

---

## Things we deliberately did NOT include in this draft

(Notes for the reviewing lawyer.)

- Arbitration clauses — explicitly not added because they're
  unenforceable for consumers in the EU.
- Class-action waivers — same.
- Force-majeure clause — let counsel decide if it adds anything
  beyond what civil law already implies.
- Indemnification by user — consumer-grade product; not appropriate.
- Auto-renewal trick language — Stripe Customer Portal already
  handles cancellation cleanly; we don't want dark patterns.
