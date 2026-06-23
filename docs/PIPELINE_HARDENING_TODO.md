# Pipeline-hardening TODO

What remains from the Phase A independent review's MUST-FIX list **on the
pipeline side**. Phase 6 landed the cheap wins (M12 refuse list, M13 date-
aware retrieval); the items here are bigger structural changes that need
their own focused commits.

Status snapshot:

| ID  | Title                              | Status            | Where it lands |
| --- | ---------------------------------- | ----------------- | -------------- |
| M2  | Eval harness (claim → expected verdict, regression-tracked) | TODO              | Phase 7 or 8   |
| M12 | Refuse list (out-of-scope categories)                       | ✅ done (Phase 6) | screener + pipeline short-circuit |
| M13 | Date-aware retrieval (resolve "today" → absolute)           | ✅ done (Phase 6) | screener system prompt now includes current date |
| M16 | Two-pass extraction (extract + dedupe + atomize)            | TODO              | Phase 7 or 8   |
| M22 | NLI citation grounding (verify investigator citations actually support the verdict) | TODO              | Phase 7 or 8   |

## M2 — eval harness

Build a small harness that runs a frozen set of seed claims with known
expected verdicts, captures the pipeline's output, and produces a
"correctness delta" against the previous run. Goal: catch regressions
when prompt or model defaults change.

Sketch:
- `apps/api/services/factcheck/eval/`
- `eval/seeds.yaml` — 25-50 hand-curated claims with expected verdict +
  source links. Mix easy/medium/hard, mix verdicts, include known-hard
  edge cases (satire, partially-true, time-sensitive).
- `eval/run.go` — runner that takes a provider + model and outputs a JSON
  report with pass/fail per seed, confidence distribution, cost.
- `make eval` to run locally; eventually wire into CI but expensive enough
  that it shouldn't block every PR — nightly is fine.

The hardest part isn't the harness — it's curating the seeds. Bias-free
curation matters; we should probably draw from existing fact-check
organizations (Snopes, FactCheck.org) with their permission/attribution.

## M16 — two-pass extraction

Today's extractor is one pass: scraped content → list of claims. Phase A
flagged that this produces redundant or over-fine claims for long inputs.
Two-pass:

1. **First pass** — extract candidate claims liberally.
2. **Second pass** — dedupe + atomize: a smaller follow-up call where the
   model sees the list and merges duplicates / splits compound claims /
   drops near-tautologies.

Cost is ~1.3× current; quality should be meaningfully better on
multi-paragraph inputs (long Instagram captions, tweet threads).

Where to land: `apps/api/services/factcheck/extract.go` (currently
inline in the pipeline scraper layer — needs to be split out first).

## M22 — NLI citation grounding

Investigators sometimes cite URLs that don't actually support the verdict
they're voting for (the URL is a real source for *something* but not for
*this claim*). M22 proposes adding an NLI (natural-language inference)
verifier that, for each citation, asks "does this source support the
claimed verdict?" and downweights citations that fail.

Implementation options:
- Cheap-model post-pass: feed (claim, verdict, snippet from source) into
  the screener model with a yes/no system prompt. Reject citations that
  come back "no". 1 cheap call per citation; cap at top-3 citations per
  investigator.
- Embeddings: cosine similarity between claim and source snippet, drop
  below a threshold. Fast + cheap but blunt.

Recommend the cheap-model post-pass. Add an `nli_verified: bool` field on
the citation persistence rows so we can see how often it actually fires.
