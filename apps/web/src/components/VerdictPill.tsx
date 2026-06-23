import { VERDICTS } from '../design/tokens'
import type { Verdict } from '../types/api'
import { cn } from '../utils/cn'

interface Props {
  verdict: Verdict
  size?: 'sm' | 'md' | 'lg'
  showDot?: boolean
  className?: string
}

export function VerdictPill({ verdict, size = 'md', showDot = true, className }: Props) {
  const meta = VERDICTS[verdict]
  const sizes = {
    sm: 'text-2xs px-2 py-0.5',
    md: 'text-xs px-2 py-1',
    lg: 'text-sm px-3 py-1.5',
  }
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-pill border font-medium uppercase tracking-wide',
        meta.bgClass, meta.borderClass, meta.textClass,
        sizes[size],
        className,
      )}
    >
      {showDot && (
        <span
          aria-hidden="true"
          className={cn(
            'inline-block h-1.5 w-1.5 rounded-full',
            meta.dotFilled ? `bg-current` : 'border border-current',
          )}
        />
      )}
      {meta.label}
    </span>
  )
}
