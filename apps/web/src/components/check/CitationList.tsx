// Citation cards. Each card shows source domain, tier glyph, title (or URL
// path), and a "vetted" badge for tier1/tier2. Stagger-in on reveal.

import { motion } from 'framer-motion'
import { ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DURATION, EASING } from '../../design/tokens'
import { TierGlyph } from '../TierGlyph'
import type { TrustTier } from '../../types/api'
import { cn } from '../../utils/cn'

export interface Citation {
  url: string
  domain: string
  title?: string
  trustTier: TrustTier
}

interface Props {
  citations: Citation[]
  className?: string
  stagger?: boolean
}

export function CitationList({ citations, className, stagger = true }: Props) {
  const { t } = useTranslation()
  if (citations.length === 0) {
    return <p className="text-sm text-fg-muted italic">No citations recorded for this verdict.</p>
  }
  return (
    <ul className={cn('space-y-2', className)}>
      {citations.map((c, i) => (
        <motion.li
          key={c.url + i}
          initial={stagger ? { opacity: 0, y: 4 } : false}
          animate={stagger ? { opacity: 1, y: 0 } : false}
          transition={{ duration: DURATION.micro, delay: i * 0.04, ease: EASING.outQuart }}
          className="flex items-start gap-3 rounded-md border border-border-subtle bg-bg-elevated px-4 py-3 hover:border-border transition-colors duration-micro"
        >
          <TierGlyph tier={c.trustTier} />
          <div className="flex-1 min-w-0">
            <div className="flex items-baseline gap-2">
              <a
                href={c.url}
                target="_blank"
                rel="noopener noreferrer nofollow"
                className="text-sm font-medium text-fg-strong hover:text-accent decoration-accent underline-offset-4 hover:underline truncate"
              >
                {c.title || c.url}
              </a>
              <ExternalLink size={12} aria-hidden="true" className="text-fg-muted shrink-0" />
            </div>
            <div className="flex items-baseline gap-2 mt-0.5">
              <span className="mono text-2xs text-fg-muted truncate">{c.domain}</span>
              {(c.trustTier === 'tier1' || c.trustTier === 'tier2') && (
                <span className="text-2xs uppercase tracking-wide text-accent">{t('check.result.vetted')}</span>
              )}
            </div>
          </div>
        </motion.li>
      ))}
    </ul>
  )
}
