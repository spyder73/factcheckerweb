// Two-layer dissent display:
//   1. Single stacked-segment SUMMARY bar — quick read for the noob
//   2. Per-investigator rows underneath — audit detail for the journalist
// Each segment / row is colored by ITS investigator's vote (not the judge's),
// so disagreement is visible at a glance.

import { useTranslation } from 'react-i18next'
import { VERDICTS } from '../../design/tokens'
import type { DissentEntry } from '../../types/api'
import { cn } from '../../utils/cn'

interface Props {
  dissent: DissentEntry[]
  className?: string
}

export function DissentBars({ dissent, className }: Props) {
  const { t } = useTranslation()
  if (dissent.length === 0) return null
  const stagger = 0.06

  // Count verdicts so the summary bar's aria-label conveys the distribution
  // — color alone isn't sufficient for AT or color-blind users.
  const counts: Record<string, number> = {}
  for (const d of dissent) counts[d.verdict] = (counts[d.verdict] ?? 0) + 1
  const summaryLabel = Object.entries(counts)
    .map(([v, c]) => `${c} ${VERDICTS[v as keyof typeof VERDICTS].shortLabel}`)
    .join(', ') + ` out of ${dissent.length} investigators`

  return (
    <div className={cn('w-full', className)}>
      <div className="flex items-baseline justify-between mb-2">
        <h3 className="text-eyebrow uppercase text-fg-muted">{t('check.result.dissent')}</h3>
        <span className="mono text-2xs text-fg-muted">N = {dissent.length}</span>
      </div>
      {/* Summary bar */}
      <div className="flex h-3 rounded-pill overflow-hidden border border-border-subtle bg-bg-sunken" role="img" aria-label={summaryLabel}>
        {dissent.map((d, i) => {
          const meta = VERDICTS[d.verdict]
          const width = 100 / dissent.length
          return (
            <div
              key={i}
              title={meta.shortLabel}
              className={cn(meta.bgClass, 'origin-left animate-dissent-bar-fill')}
              style={{ width: `${width}%`, animationDelay: `${i * stagger}s` }}
            />
          )
        })}
      </div>
      {/* Per-investigator rows */}
      <ul className="mt-4 space-y-1.5">
        {dissent.map((d, i) => {
          const meta = VERDICTS[d.verdict]
          return (
            <li key={i} className="flex items-center gap-3 text-sm">
              <span className="mono text-2xs w-12 text-fg-muted">#{d.position + 1}</span>
              <span className="w-28 text-fg-subtle truncate" title={d.style}>{d.style}</span>
              <div className="flex-1 h-2 bg-bg-sunken rounded-pill overflow-hidden">
                <div
                  className={cn(meta.bgClass, 'h-full origin-left animate-dissent-bar-fill')}
                  style={{ width: `${Math.max(8, d.confidence * 100)}%`, animationDelay: `${i * stagger}s` }}
                />
              </div>
              <span className={cn('w-24 text-right text-xs font-medium', meta.textClass)}>{meta.shortLabel}</span>
              <span className="tabular w-10 text-right text-2xs text-fg-muted">{Math.round(d.confidence * 100)}%</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
