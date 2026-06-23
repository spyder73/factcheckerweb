# How Alethea works

A long-form explanation of the pipeline, written for someone who wants to
trust (or not trust) the verdict on their screen. If you only want a
30-second version, see the diagram on the homepage.

---

## The promise

**Paste anything. Get a verdict you can trace.** Every verdict on Alethea
shows you exactly what we looked at, who voted which way, and which
sources we used — so you can disagree with us on the evidence, not on
faith. We are deliberately skeptical: when the evidence is thin, the
verdict says "not enough evidence," never a confident guess.

---

## The pipeline

1. **Scrape.** When you paste a URL, we fetch the page and pull out the
   text + any attached images. For Instagram we use a dedicated service
   (the public API is locked down); for X/TikTok/YouTube/Facebook and
   generic web pages we use OpenGraph tags + a server-side fetch with
   SSRF protections (we resolve the DNS to an IP, check it isn't private,
   then re-check the same IP when we connect — so a malicious server
   can't redirect us to your internal network).
2. **Media analysis.** Every attached image goes through a vision model
   in parallel. We don't paste the raw base64 into the next stage — we
   summarize each image to text first. This keeps the prompt under the
   model's context window and means image evidence is treated like any
   other source.
3. **Claim extraction.** A cheap model reduces everything we collected
   (text + image summaries) into a list of atomic, testable claims. Long
   posts often produce 3–8 claims; one-liners often produce one. If the
   input is opinion, prediction, personal experience, or humor, the
   pipeline refuses to fact-check it and labels why — we don't pretend
   we can verify things we can't.
4. **Screening.** For each claim, a screening pass produces an early
   hint: how hard is this, what's the rough verdict the model thinks,
   what questions would resolve it. The screener also resolves relative
   dates ("today", "this year") into absolute references so retrieval
   isn't searching for stale terms.
5. **Retrieval.** We issue searches against Brave + Tavily, then merge
   and de-duplicate by URL. Curated sources (Reuters, AP, peer-reviewed
   journals, official statistics agencies, etc.) get reranked to the top
   of the pool. If we can't find at least two distinct domains for a
   claim, the final confidence is capped.
6. **Investigators.** A fan-out of N independent agents (each given a
   different "investigation style") gets the same source pool and is
   asked to reason about the claim. They must cite sources by ID; they
   can't invent URLs.
7. **Judge.** A stronger model reads all N investigator reports + the
   source pool and produces the final verdict, summary, reasoning, and
   citations. The judge sees the dissent — when investigators disagree,
   the judge has to say so explicitly.
8. **Skeptical floor.** If the final confidence is under 0.65, the
   verdict is automatically demoted to "not enough evidence" regardless
   of what the judge said. We'd rather say "we don't know" than gamble.
9. **Content hash.** Every verdict is hashed against its inputs (prompt
   version, normalized claim, sorted source URLs, sorted transcript
   hashes). Two checks of the same content will produce the same hash —
   useful for non-repudiation and for spotting drift when we change
   prompts.

---

## What "verified" really means

We use deliberately conservative labels:

| Label                 | What it means                                                                |
| --------------------- | ---------------------------------------------------------------------------- |
| WELL-SUPPORTED        | Multiple independent reputable sources confirm this.                         |
| PARTIALLY TRUE        | Some parts hold up; others don't. Read the details.                          |
| MISLEADING            | Technically possible but framed to deceive.                                  |
| CONTRADICTED          | Sources contradict the claim. (We avoid the word "false" because absolute claims often turn out to have edge cases — "contradicted by the evidence we found" is the more honest framing.) |
| NOT ENOUGH EVIDENCE   | We couldn't find enough to judge — including any time the skeptical floor demoted us.|
| SATIRE                | This is humor or parody. Treat as such.                                      |
| NO CLAIM              | There isn't a factual claim to verify, or it's out of scope.                 |

---

## What Alethea is NOT

- **Not a final authority.** Read the cited sources. We surface them so
  you can judge for yourself. If you disagree with our verdict, the
  evidence is right there to argue with.
- **Not an opinion-arbiter.** We refuse to fact-check value judgments
  ("X is the best", "Y was a mistake") and clearly label them as such.
- **Not real-time.** A verdict is a snapshot. If the underlying facts
  change (an event unfolds, a source retracts), the verdict needs to
  change too. We're building a correction-chain system so older verdicts
  can be linked to newer ones rather than silently overwritten.
- **Not infallible.** We will make mistakes. When we do, the corrections
  system above is how we own them publicly.

---

## What's open

The pipeline code, prompts, source list, trust tiers, and verdict
templates are all in [the public repo](https://github.com/alethea-app).
Anyone can read what we ask the model, what sources we trust, and how we
combine investigator outputs. Anyone can self-host it.

The only things that aren't public are:
- Individual users' BYOK keys (AES-GCM-encrypted at rest; we can't read them).
- The current month's financial breakdown until the close of month (it's
  posted on `/economics` once the month closes).
- User-identifying data (history, journalist application contents).

---

## Limits we'll be transparent about

- **Search coverage**: we cover English well, German tolerably, and
  Romance-language European news partially. We don't cover non-Latin
  scripts well yet.
- **Image accuracy**: vision models can hallucinate text from images;
  treat OCR'd text from screenshots with skepticism.
- **Recency**: very-recent events (breaking-news < 6 hours) are often
  under-sourced. Confidence will be lower on those.
- **Niche or technical claims**: if the relevant evidence is behind
  paywalls (medical journals, paid datasets), our retrieval misses it.

If a verdict feels wrong, write us through [security@alethea.example] or
file an issue with the verdict ID — we read every report.
