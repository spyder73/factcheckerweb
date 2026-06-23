# Journalist program

A free upgrade to Alethea Plus for working journalists. Goal: get the
people who most need traceable verdicts to use the tool routinely, and
get their feedback in return.

## What you get

- Unlimited fact-checks (subject to the global cost circuit-breaker —
  currently $200/day platform-wide).
- Multi-agent (N=5) consensus verdicts.
- Priority queue when the platform is busy.
- Export-as-PDF + ClaimReview JSON-LD download for embedding in your
  CMS.
- A `journalist` badge on any public profile (off by default).

## Eligibility

Working journalists in any country, including freelance. Specifically:

- Published bylines in news outlets (at least 3 in the past 12 months).
- OR an editorial role at a recognized newsroom.
- OR a press credential issued by a recognized body.

We're deliberately broad here. Substack writers, podcast hosts,
independent investigators, and academic researchers who publish on news
topics are welcome. Activists running advocacy organizations are not —
that's a different category and we don't want the badge to launder
opinion as reporting.

## How to apply

In your Alethea account: Settings → Journalist program → Submit
application. We ask for:

- Full name (real name, not a pseudonym — we may need to verify with
  your outlet).
- Outlet name + URL.
- Country.
- Beat (optional).
- 1-3 byline URLs from the past 12 months.
- Short bio (200 chars).

Your application is stored in the `journalist_applications` table and
reviewed by a maintainer within 5 business days.

## Review criteria

We will approve if:

1. The byline URLs resolve to articles you authored (we click through
   and check).
2. The outlets are recognizable (a domain we know, or a domain we can
   verify is a real publication via OpenGraph + corrections policy).
3. You haven't been previously suspended from the program.

We will reject (with a reason) if:

- The bylines don't exist or aren't by the applicant.
- The "outlet" is a personal blog with no editorial process. We don't
  judge content quality; we judge whether there's a journalism
  structure around it.
- The application looks duplicate / fake.

## What we ask of you

In exchange for the free Plus tier:

- **Use it on real work.** Not a hard requirement, but the program is
  funded by donations and we want to direct that subsidy at people who
  publish.
- **Tell us when we're wrong.** Email `corrections@alethea.example`
  with the verdict ID and what was wrong. We will issue a public
  correction.
- **Don't impersonate.** Use of someone else's identity to gain access
  permanently bans you from the program.

We don't require:

- Attribution on stories you use Alethea for. It's a tool; you don't
  need to credit your spell-checker.
- Exclusivity. Use other fact-checking tools too — competition is good.

## Suspension / removal

We will suspend the badge (and revert your account to standard Plus
billing terms — i.e. you'd need to start paying, or downgrade) for:

- Confirmed identity misrepresentation in the application.
- Using the badge to harass or impersonate other journalists.
- Mass-automated abuse of the unlimited-checks privilege (the global
  circuit-breaker catches this; we just close the loophole when we
  see it).

Suspension is communicated by email with a reason. You can appeal.

## Privacy

Your application is stored only as long as your account exists. If you
delete your account (Settings → Privacy → Delete my account), the
journalist application row is deleted along with everything else.

Approved journalists are NOT a public list — we don't publish who's in
the program. The badge on your verdict pages is opt-in.

## We're building this slowly

This is a v1 program. We expect to learn things like:

- Whether the byline-URL criterion is too strict (probably yes for
  some legitimate underground reporting).
- Whether the badge causes verdict-laundering (someone using
  Alethea's "journalist-checked" badge as social proof for a story
  the badge doesn't endorse — we'd want clearer language about what
  the badge actually means).
- Whether to add a paid "newsroom" tier on top of this for outlets
  that want to embed verdicts at scale.

If you have feedback, email `journalism@alethea.example` or open an
issue.
