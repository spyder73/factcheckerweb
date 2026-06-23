# Source policy

How Alethea decides which sources go into the curated trust registry,
how tiers are assigned, and how the list can be challenged.

Companion to [`trust-policy.md`](trust-policy.md), which explains how the
trust tiers feed into the pipeline.

## What's in the registry

The curated source registry is a list of **domains**, each tagged with:

- **Trust tier** (1, 2, 3, or unknown). See `trust-policy.md` for the
  definitions.
- **Country of primary editorial control** (where applicable).
- **Category** — news wire, daily newspaper, broadcaster, journal,
  government agency, NGO, etc.
- **Active / deprecated** — we never delete, only deprecate. Deprecated
  domains keep their historical assignment so old verdicts remain
  reproducible.

The registry is stored in Postgres (`sources` table) and exposed via
`/api/sources`. The seed list lives at
`apps/api/db/migrations/00012_seed_sources.sql`.

## Tier 1 criteria

A domain qualifies for tier 1 only when ALL of the following are true:

1. **Public editorial standards.** A published policy describing how
   stories are sourced, fact-checked, and corrected.
2. **Public corrections policy.** A documented process for issuing
   corrections, and a visible track record of doing so.
3. **Named bylines.** Real journalists' names on stories, not "Staff."
4. **No paywall on corrections.** Corrections must be readable without
   a subscription, even if the original is paywalled.
5. **Multi-source on hard stories.** A pattern of citing primary
   documents rather than other outlets.
6. **Operational independence.** Not directly owned by the entities
   they primarily cover (e.g. a state-owned outlet covering its own
   government doesn't qualify for tier 1 on stories about that
   government).

Examples currently in tier 1: Reuters, AP, AFP, BBC World News
(English), Nature, NEJM, ECDC, WHO press, OECD statistics, US BLS,
Eurostat, IPCC reports, ICJ rulings, court records (PACER, BAILII).

## Tier 2 criteria

Mainstream outlets with newsrooms, named bylines, public corrections
policies, and a track record. They don't fully meet tier 1 only because
they're a single outlet (not a wire) and have an identifiable editorial
slant.

Examples: NY Times, WaPo, FT, Guardian, Süddeutsche Zeitung, Die Zeit,
NRC, El País, Le Monde, NPR, ABC (Australia), CBC, NHK, Aftenposten.

## Tier 3 criteria

Specialist outlets, smaller but cited papers, advocacy organizations
that publish their data and methodology transparently. They're useful
evidence but the user should know what they are.

Examples: Pew Research, ProPublica, Bellingcat, MIT Technology Review,
Centre for Strategic and International Studies, Tax Policy Center,
peer-reviewed journals outside the top 5 in a field.

## Why "unknown" exists

The registry can never cover everything. When a source domain isn't in
the registry, investigators can still cite it — it just shows as
"Unknown source" in the UI and the judge weights it less heavily.

Over time, the registry grows: we add domains we keep seeing in the
unknown bucket and that pass the criteria above. The seed list is
intentionally small (~50 domains) so it's manageable to maintain.

## Who decides?

For v1: the project maintainer. Decisions are made publicly via a PR to
`apps/api/db/migrations/0001N_sources.sql` with a one-paragraph
justification per domain. This is obviously a single point of failure
and we'll move to a small editorial board once we have one.

## Challenging the list

If you think a domain is misclassified — too high or too low — open a
GitHub issue with the domain, the tier you think is right, and the
evidence. We respond to every challenge publicly.

Specifically WELCOME challenges:

- Showing that a tier 1 outlet failed one of the tier 1 criteria
  (e.g. their corrections policy is broken). Concrete examples beat
  general critiques.
- Pointing out an outlet missing from the registry that meets the
  criteria above.
- Flagging country-specific quality issues we'd otherwise miss
  (e.g. an outlet that's reputable in its country but unknown to us).

Specifically NOT effective:

- "Outlet X is biased" without showing how it failed a specific
  criterion. Bias and quality are different axes.
- "Add my blog/Substack." Individual writers don't qualify regardless
  of quality; the registry is at the domain level.

## What we will never do

- Pay-for-inclusion: no outlet can buy a tier, and no advertising
  relationship affects the registry.
- Hide criteria: if we change the tier definitions, the change is in
  the public git history.
- Quietly demote outlets after a public dispute. Demotions go through
  the same PR process as additions.

## What we will probably do later

- Per-claim tier overrides. An outlet might be tier 1 on politics but
  not on science. The current schema doesn't support this; future
  schema will add a `(domain, topic, tier)` overlay.
- A small editorial board to take individual maintainer judgment out
  of the loop, once there's funding to compensate them.
- A separate "primary source" bucket for things like court rulings
  and bills — they're more authoritative than even tier 1 reporting
  about them, but currently shoehorned into tier 1.
