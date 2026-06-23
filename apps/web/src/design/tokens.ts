// Single source of truth for design decisions that classes can't express:
// motion budgets, verdict metadata, tier glyphs. Components MUST import
// from here — no hardcoded hex, no inline durations, no scattered verdict
// color maps.

import type { Verdict, TrustTier } from '../types/api'

// ---------- VERDICT METADATA ----------

export interface VerdictMeta {
  id: Verdict
  label: string
  shortLabel: string
  description: string
  textClass: string
  bgClass: string
  borderClass: string
  dotFilled: boolean
  motionTier: 'narrative' | 'none'
  legendOrder: number
}

export const VERDICTS: Record<Verdict, VerdictMeta> = {
  verified: {
    id: 'verified', label: 'VERIFIED', shortLabel: 'Verified',
    description: 'Claim is supported by tier-1 sources.',
    textClass: 'text-verdict-verified', bgClass: 'bg-verdict-verified-bg', borderClass: 'border-verdict-verified',
    dotFilled: true, motionTier: 'narrative', legendOrder: 1,
  },
  partially_true: {
    id: 'partially_true', label: 'PARTIALLY TRUE', shortLabel: 'Partially true',
    description: 'Some parts true, some not. Read both halves.',
    textClass: 'text-verdict-partially-true', bgClass: 'bg-verdict-partially-true-bg', borderClass: 'border-verdict-partially-true',
    dotFilled: false, motionTier: 'narrative', legendOrder: 2,
  },
  misleading: {
    id: 'misleading', label: 'MISLEADING', shortLabel: 'Misleading',
    description: 'Technically true, materially deceptive.',
    textClass: 'text-verdict-misleading', bgClass: 'bg-verdict-misleading-bg', borderClass: 'border-verdict-misleading',
    dotFilled: false, motionTier: 'narrative', legendOrder: 3,
  },
  false: {
    id: 'false', label: 'FALSE', shortLabel: 'False',
    description: 'Contradicted by tier-1 sources.',
    textClass: 'text-verdict-false', bgClass: 'bg-verdict-false-bg', borderClass: 'border-verdict-false',
    dotFilled: true, motionTier: 'narrative', legendOrder: 4,
  },
  unverifiable: {
    id: 'unverifiable', label: 'UNVERIFIABLE', shortLabel: 'Unverifiable',
    description: "Not enough evidence either way. The skeptical default.",
    textClass: 'text-verdict-unverifiable', bgClass: 'bg-verdict-unverifiable-bg', borderClass: 'border-verdict-unverifiable',
    dotFilled: false, motionTier: 'none', legendOrder: 5,
  },
  satire: {
    id: 'satire', label: 'SATIRE', shortLabel: 'Satire',
    description: 'Marked or obviously not literal.',
    textClass: 'text-verdict-satire', bgClass: 'bg-verdict-satire-bg', borderClass: 'border-verdict-satire',
    dotFilled: true, motionTier: 'narrative', legendOrder: 6,
  },
  no_claim: {
    id: 'no_claim', label: 'NO CLAIM', shortLabel: 'No claim',
    description: 'Post contains no factual claim to check.',
    textClass: 'text-verdict-no-claim', bgClass: 'bg-verdict-no-claim-bg', borderClass: 'border-verdict-no-claim',
    dotFilled: true, motionTier: 'none', legendOrder: 7,
  },
}

export const VERDICTS_BY_LEGEND_ORDER: VerdictMeta[] = Object.values(VERDICTS).sort(
  (a, b) => a.legendOrder - b.legendOrder
)

// ---------- TRUST TIER ----------

export interface TierMeta {
  tier: TrustTier
  glyph: string
  label: string
  shortLabel: string
  description: string
}

// Diamond-glyph convention borrowed from press-freedom indices.
// Maps the API's string tier into the integer tier expected here.
export const TIER_MAP: Record<TrustTier, number> = {
  tier1: 1, tier2: 2, tier3: 3, unknown: 4,
}

export const TIERS: Record<number, TierMeta> = {
  1: { tier: 'tier1', glyph: '◆◆◆◆', label: 'Tier 1 — Primary', shortLabel: 'Tier 1',
       description: 'Primary sources: government data, peer-reviewed research, court records, official statements.' },
  2: { tier: 'tier2', glyph: '◆◆◆◇', label: 'Tier 2 — Vetted news', shortLabel: 'Tier 2',
       description: 'Established news with corrections policies and editorial standards.' },
  3: { tier: 'tier3', glyph: '◆◆◇◇', label: 'Tier 3 — Specialist', shortLabel: 'Tier 3',
       description: 'Specialist or known-good outlets with limited general vetting.' },
  4: { tier: 'unknown', glyph: '◆◇◇◇', label: 'Not vetted', shortLabel: 'Unvetted',
       description: 'Not in our curated registry. Treat as context only.' },
}

// ---------- MOTION BUDGET ----------

export const EASING = {
  outQuart: [0.25, 1, 0.5, 1] as const,
  inOutQuart: [0.76, 0, 0.24, 1] as const,
} as const

export const DURATION = {
  micro: 0.12,
  macro: 0.24,
  narrative: 0.6,
} as const

export interface MotionPolicy {
  allowNarrative: boolean
  reducedMotion: boolean
}

export function resolveMotionPolicy(
  verdict: Verdict,
  skepticalFallback: boolean,
  reducedMotion: boolean,
): MotionPolicy {
  const verdictAllowsNarrative = VERDICTS[verdict].motionTier === 'narrative'
  return {
    allowNarrative: verdictAllowsNarrative && !skepticalFallback && !reducedMotion,
    reducedMotion,
  }
}

// ---------- SSE PIPELINE LABELS ----------

export const STAGE_LABEL: Record<string, string> = {
  init: 'Starting',
  resolve: 'Fetching the page',
  media: 'Reading the images',
  extract: 'Pulling out testable claims',
  claims: 'Claims extracted',
  screen: 'Quick screen',
  retrieval: 'Searching the open web',
  retrieval_failed: 'Search unavailable',
  pool: 'Source pool ready',
  fanout: 'Investigators dispatched',
  investigator: 'Investigator reasoning',
  judge: 'Synthesizing the verdict',
  cache_hit: 'Served from cache',
  claim_done: 'Claim done',
  byok_fallback: 'BYOK fell back to pool',
  done: 'Done',
  error: 'Error',
}
