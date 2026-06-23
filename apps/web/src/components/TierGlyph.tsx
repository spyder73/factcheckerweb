// Diamond-glyph trust-tier badge. Screen-reader text translates the glyph
// to a human description.
import { TIER_MAP, TIERS } from '../design/tokens'
import type { TrustTier } from '../types/api'
import { cn } from '../utils/cn'

interface Props {
  tier: TrustTier
  className?: string
}

export function TierGlyph({ tier, className }: Props) {
  const t = TIER_MAP[tier] ?? 4
  const meta = TIERS[t]
  return (
    <span
      title={meta.label}
      className={cn('mono text-xs text-fg-muted', className)}
      aria-label={meta.label}
    >
      {meta.glyph}
    </span>
  )
}
