// The actual verdict reveal — ring + dissent + summary + per-claim breakdown
// + citations + intent. The single place a "narrative" reveal can fire.

import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { AlertCircle, Database, Eye } from 'lucide-react'
import { DURATION, EASING, VERDICTS } from '../../design/tokens'
import type { CheckDonePayload, TrustTier } from '../../types/api'
import { VerdictRing } from './VerdictRing'
import { DissentBars } from './DissentBars'
import { CitationList } from './CitationList'
import type { Citation } from './CitationList'
import { VerdictPill } from '../VerdictPill'
import { cn } from '../../utils/cn'
import { useReducedMotion } from '../../hooks/useReducedMotion'

interface Props {
  result: CheckDonePayload
}

export function ResultBlock({ result }: Props) {
  const { t } = useTranslation()
  const reduced = useReducedMotion()
  const cached = result.claims.length > 0 && result.claims.every((c) => c.cache_hit)
  const skeptical = result.claims.some((c) => c.skeptical_fallback)

  return (
    <div className="space-y-12">
      {/* Top: ring + dissent + summary, three-column on lg */}
      <section
        aria-label="Final verdict"
        className={cn('grid gap-8 lg:grid-cols-[auto_1fr] items-start')}
      >
        <div className="flex justify-center lg:justify-start">
          <VerdictRing
            verdict={result.overall_verdict}
            confidence={result.overall_confidence}
            skepticalFallback={skeptical}
          />
        </div>
        <div className="space-y-6">
          <div>
            <div className="flex items-center gap-3 mb-3">
              <VerdictPill verdict={result.overall_verdict} size="lg" />
              {cached && <span className="text-xs text-fg-muted">{t('check.result.cached')}</span>}
            </div>
            {/* Action line — the highest-priority Phase A finding (M1):
                non-expert readers see a verb-first instruction before the prose. */}
            <p className={`text-lg font-semibold mb-3 ${VERDICTS[result.overall_verdict].textClass}`}>
              {VERDICTS[result.overall_verdict].actionLine}
            </p>
            <p className="text-base text-fg-subtle max-w-measure">{result.summary}</p>
          </div>

          {skeptical && (
            <motion.div
              initial={reduced ? false : { opacity: 0, y: 4 }}
              animate={reduced ? false : { opacity: 1, y: 0 }}
              transition={{ duration: DURATION.macro, ease: EASING.outQuart }}
              className="flex items-start gap-3 rounded-md border border-verdict-partially-true bg-verdict-partially-true-bg px-4 py-3"
              role="note"
            >
              <AlertCircle size={16} className="text-verdict-partially-true mt-0.5 shrink-0" aria-hidden="true" />
              <div className="text-sm text-fg">{t('check.result.skeptical')}</div>
            </motion.div>
          )}

          {/* Per-claim sections */}
          {result.claims.length > 1 && (
            <div className="space-y-6 pt-4 border-t border-border-subtle">
              <h2 className="text-eyebrow uppercase text-fg-muted">{t('check.result.claims')}</h2>
              {result.claims.map((c, i) => (
                <article key={c.claim.id} className="space-y-3">
                  <div className="flex items-baseline gap-3">
                    <span className="mono text-2xs text-fg-muted">claim_{(i + 1).toString().padStart(2, '0')}</span>
                    <VerdictPill verdict={c.final.verdict} size="sm" />
                  </div>
                  <p className="text-fg">{c.claim.raw_text}</p>
                  <p className="text-sm text-fg-subtle prose-measure">{c.final.summary}</p>
                </article>
              ))}
            </div>
          )}
        </div>
      </section>

      {/* Dissent — always visible, placed BEFORE the citations to physically
          encode "see disagreement before agreement". */}
      {result.claims.length > 0 && result.claims[0].dissent.length > 1 && (
        <section aria-labelledby="dissent-heading" className="border-t border-border-subtle pt-8">
          <h2 id="dissent-heading" className="sr-only">{t('check.result.dissent')}</h2>
          <DissentBars dissent={result.claims[0].dissent} />
        </section>
      )}

      {/* Citations — flatten across claims for the overall list */}
      <section aria-labelledby="citations-heading" className="border-t border-border-subtle pt-8">
        <h2 id="citations-heading" className="text-eyebrow uppercase text-fg-muted mb-4 flex items-center gap-2">
          <Database size={14} aria-hidden="true" />
          {t('check.result.citations')}
        </h2>
        <CitationList citations={collectCitations(result)} />
      </section>

      {/* Intent panel — separately labeled as interpretation */}
      {result.claims.length > 0 && result.claims[0].final.intent_interpretation && (
        <section
          aria-labelledby="intent-heading"
          className="border border-accent/30 bg-accent-bg/40 rounded-lg px-6 py-5"
        >
          <div className="flex items-baseline gap-3 mb-2">
            <Eye size={14} aria-hidden="true" className="text-accent" />
            <h2 id="intent-heading" className="text-eyebrow uppercase text-accent">{t('check.result.intent')}</h2>
            <span className="mono text-2xs text-fg-muted">interpretive, not factual</span>
          </div>
          <p className="text-fg prose-measure">{result.claims[0].final.intent_interpretation}</p>
        </section>
      )}

      {/* Provenance footer */}
      <section className="border-t border-border-subtle pt-4">
        <p className="mono text-2xs text-fg-muted">
          check_{result.id.slice(0, 8)} · {result.elapsed_ms}ms ·{' '}
          {result.byok_used ? 'byok' : 'pool'}
          {result.byok_fellback && ' · fellback'}
          {result.tokens_in !== undefined && ` · ${result.tokens_in + (result.tokens_out ?? 0)} tokens`}
          {result.cost_micros !== undefined && ` · €${(result.cost_micros / 1_000_000).toFixed(4)}`}
        </p>
      </section>
    </div>
  )
}

function collectCitations(r: CheckDonePayload): Citation[] {
  // The pipeline streams citations grouped by agent_run; for the v1 display
  // we de-dup by URL + take the highest trust tier seen for each.
  const byURL = new Map<string, Citation>()
  for (const claim of r.claims) {
    for (const url of claim.final.cited_urls ?? []) {
      const existing = byURL.get(url)
      if (existing) continue
      byURL.set(url, {
        url,
        domain: extractDomain(url),
        trustTier: 'unknown' as TrustTier,
      })
    }
  }
  return Array.from(byURL.values())
}

function extractDomain(u: string): string {
  try {
    return new URL(u).hostname.replace(/^www\./, '')
  } catch {
    return u
  }
}
