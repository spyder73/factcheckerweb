// Mobile design tokens — kept deliberately small and matched to the
// web tokens (apps/web/src/design/tokens.ts) so verdict colors stay
// consistent across surfaces. Dark-first; light mode mirrors web.

import type { Verdict } from '@alethea/shared-types'

export const COLORS = {
  bg: '#0b0b0d',
  bgElevated: '#141418',
  border: '#2a2a30',
  borderSubtle: '#1c1c22',
  fg: '#e8e8ec',
  fgMuted: '#9999a3',
  fgSubtle: '#6f6f78',
  accent: '#7c5cff',
  accentFg: '#ffffff',
}

export const VERDICT_COLORS: Record<Verdict, string> = {
  verified: '#3fb950',
  partially_true: '#d29922',
  misleading: '#f0883e',
  false: '#f85149',
  unverifiable: '#8b949e',
  satire: '#a371f7',
  no_claim: '#6f6f78',
}

// M1 verdict-copy rewrite — keep this in sync with the web tokens.
export const VERDICT_LABEL: Record<Verdict, string> = {
  verified: 'WELL-SUPPORTED',
  partially_true: 'PARTIALLY TRUE',
  misleading: 'MISLEADING',
  false: 'CONTRADICTED',
  unverifiable: 'NOT ENOUGH EVIDENCE',
  satire: 'SATIRE',
  no_claim: 'NO CLAIM',
}

export const VERDICT_ACTION_LINE: Record<Verdict, string> = {
  verified: 'Multiple reputable sources confirm this.',
  partially_true: 'Some parts hold up; others don\'t. Read the details.',
  misleading: 'Technically possible but framed to deceive. Be careful.',
  false: "Don't share this — sources contradict it.",
  unverifiable: 'We couldn\'t find enough to judge. Don\'t share without checking yourself.',
  satire: 'This is humor or parody. Treat as such.',
  no_claim: 'No factual claim to verify.',
}
