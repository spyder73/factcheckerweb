# Trust policy

How Alethea decides which sources count as "trusted," how the
multi-agent pipeline reaches a verdict, and what we promise about
non-repudiation. Companion to `source-policy.md`.

## Source trust tiers

We classify every source domain (not specific URL) into one of four tiers:

- **Tier 1 — Primary reference.** Wire services with publicly documented
  editorial standards, peer-reviewed journals, official statistics
  agencies, court records, primary government data. Examples: Reuters,
  AP, AFP, Nature, ECDC, OECD, IPCC reports, BLS statistics.
- **Tier 2 — Established mainstream.** Outlets with newsrooms, named
  bylines, public corrections policies, and a track record of issuing
  retractions when wrong. Examples: BBC News, NY Times, FT, Guardian,
  Süddeutsche Zeitung, Le Monde, NPR.
- **Tier 3 — Verifiable but bias-prone.** Specialist outlets, smaller
  but cited papers, advocacy organizations that publish their data
  transparently. We use them but flag them as "Tier 3" to the user.
- **Unknown.** Anything not in our registry. Investigators can still
  cite these but the judge treats them as supporting, not conclusive.

The full source list with assigned tiers lives at `/api/sources` and is
mirrored on the `/sources` page on the site. The registry is
human-curated and the criteria are in [`source-policy.md`](source-policy.md).

## How verdicts are reached

Phase 2+ pipeline (see [`how-it-works.md`](how-it-works.md) for the long
form):

1. **Shared retrieval.** All investigators see the same source pool. We
   reranked tier 1+2 sources to the top.
2. **Independent reasoning.** N investigators (3 for BYOK / Free
   accounts; 5 for Plus / Admin) work independently — they each see the
   pool but not each other's drafts.
3. **Judge synthesis.** A stronger model reads all N reports plus the
   source pool and produces the final verdict + reasoning. The judge is
   required to explicitly acknowledge dissent when investigators
   disagree.
4. **Skeptical floor.** If the judge's confidence is below 0.65, the
   verdict is automatically demoted to "not enough evidence." We log
   what the judge said pre-floor so a journalist can audit the
   demotion.

### Domain diversity

If the source pool spans fewer than 2 distinct domains, the final
confidence is capped at 0.55 — even if every investigator agreed. One
domain isn't enough triangulation, regardless of its tier.

### Tier weighting

We do not multiply confidence by tier. A tier 1 source is meaningfully
better evidence than a tier 3 source, but the model decides how much
weight to give each — we just surface the tier to it in the prompt.
This is deliberate: hard-coded tier weights bake our preferences into
math, which is harder to argue with than a transparent prompt.

## Non-repudiation

Every verdict has a `content_hash`: SHA-256 of
`(prompt_version | normalized_claim | sorted_source_urls | sorted_transcript_hashes)`.

That hash is stored on the verdict row and on the public verdict page.
Implications:

- Two checks of the same content produce the same hash.
- If we ever change a prompt or model, you can see (via the prompt
  version in the hash inputs) that the verdict was produced by a
  different pipeline. We don't silently update old verdicts.
- We can prove that a specific verdict came from a specific pipeline
  configuration at a specific point in time.

## Corrections

We will make mistakes. When we do:

1. A new verdict is generated and stored.
2. The old verdict's `superseded_by` column points at the new one (and
   the old verdict is preserved — never deleted).
3. The verdict page shows a "Updated YYYY-MM-DD — newer verdict
   available" banner with the reason for the correction.
4. The correction is logged in `audit_log` and shows up on the
   transparency page.

We're committing to this publicly because the alternative (silently
deleting bad verdicts) is the thing fact-checkers do that destroys
their credibility the fastest.

## Refusal categories

The screening pass refuses to issue a verdict (returns "out of scope")
for:

- **Opinions** — value judgments, aesthetic preferences.
- **Predictions** — claims about the future.
- **Personal experience** — claims about the speaker's own private
  experience.
- **Claims about private individuals** — when the subject is not a
  public figure and the claim concerns their private life.
- **Religion / ideology** — metaphysical or doctrinal claims that
  aren't testable against evidence.
- **Obvious humor / sarcasm** — treating a joke as a literal factual
  claim would miss the point.

Refusals are not failures — they're a feature. We'd rather say "this
isn't the kind of thing we can verify" than fabricate confidence on
something we can't actually evaluate.

## Conflicts of interest

We are an independent non-profit (or solo developer + donations, in
v1). We are not commercially affiliated with any publisher, platform,
or political organization. If that changes, we'll disclose it on
`/about` and on every verdict footer.

Investigators run on commercial LLMs (Anthropic, OpenAI, Mistral,
others via OpenRouter). The choice of model is a known risk: models
have their own biases. We mitigate this with:

- Multi-vendor fan-out where available (so a single model's bias
  doesn't dominate).
- A documented prompt that explicitly requires the model to cite
  sources, not opinions.
- The skeptical floor.

We do not currently fine-tune our own model. If we do, that fine-tuning
data will be open.

## Reporting a bad verdict

Email `security@alethea.example` with the verdict ID (the URL on the
verdict page) and what you think went wrong. We read every report. If
the verdict is corrected as a result of your report, your name (or
pseudonym) goes in the acknowledgments unless you ask us not to.
