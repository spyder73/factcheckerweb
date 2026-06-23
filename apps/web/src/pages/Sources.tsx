import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { listSources } from '../api/sources'
import type { Source } from '../types/api'
import { TierGlyph } from '../components/TierGlyph'
import { TIERS, TIER_MAP } from '../design/tokens'

const CATEGORIES = [
  'all', 'wire', 'newspaper', 'magazine', 'factcheck',
  'academic', 'government', 'specialist', 'encyclopedia', 'primary',
]
const TIERS_FILTER = ['all', 'tier1', 'tier2', 'tier3', 'unknown'] as const

export default function Sources() {
  const { t } = useTranslation()
  const [sources, setSources] = useState<Source[] | null>(null)
  const [loadErr, setLoadErr] = useState<string | null>(null)
  const [cat, setCat] = useState('all')
  const [tier, setTier] = useState<(typeof TIERS_FILTER)[number]>('all')

  useEffect(() => {
    let cancelled = false
    listSources().then(
      (r) => { if (!cancelled) setSources(r.sources) },
      (e) => { if (!cancelled) setLoadErr(e instanceof Error ? e.message : 'failed to load') },
    )
    return () => { cancelled = true }
  }, [])

  const filtered = useMemo(
    () => (sources ?? []).filter((s) => (cat === 'all' || s.category === cat) && (tier === 'all' || s.trustTier === tier)),
    [sources, cat, tier],
  )

  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-3xl font-bold mb-2">{t('sources.title')}</h1>
      <p className="text-fg-subtle prose-measure mb-8">{t('sources.subtitle')}</p>

      <div role="group" aria-label={t('sources.filters.category')} className="flex flex-wrap items-center gap-2 mb-6">
        <span className="text-eyebrow uppercase text-fg-muted mr-2">{t('sources.filters.category')}</span>
        {CATEGORIES.map((c) => (
          <button key={c} type="button" onClick={() => setCat(c)} aria-pressed={cat === c}
            className={`rounded-pill border px-3 py-1 text-sm ${cat === c ? 'border-accent text-accent bg-accent-bg' : 'border-border-subtle text-fg-muted hover:text-fg'}`}>
            {c === 'all' ? t('sources.filters.all') : c}
          </button>
        ))}
      </div>
      <div role="group" aria-label={t('sources.filters.tier')} className="flex flex-wrap items-center gap-2 mb-8">
        <span className="text-eyebrow uppercase text-fg-muted mr-2">{t('sources.filters.tier')}</span>
        {TIERS_FILTER.map((tt) => (
          <button key={tt} type="button" onClick={() => setTier(tt)} aria-pressed={tier === tt}
            className={`rounded-pill border px-3 py-1 text-sm ${tier === tt ? 'border-accent text-accent bg-accent-bg' : 'border-border-subtle text-fg-muted hover:text-fg'}`}>
            {tt === 'all' ? t('sources.filters.all') : TIERS[TIER_MAP[tt]].shortLabel}
          </button>
        ))}
      </div>

      {sources === null && !loadErr && (
        <p className="text-fg-muted py-8">Loading curated sources…</p>
      )}
      {loadErr && (
        <p role="alert" className="text-verdict-false py-8">Couldn't load sources: {loadErr}</p>
      )}
      {sources !== null && (
        <div className="overflow-x-auto -mx-4 lg:mx-0">
          <table className="w-full text-sm min-w-[640px]">
            <caption className="sr-only">Curated sources Alethea recognizes by default</caption>
            <thead className="border-b border-border-subtle text-eyebrow uppercase text-fg-muted">
              <tr>
                <th scope="col" className="text-left py-2 pr-4 pl-4 lg:pl-0">Tier</th>
                <th scope="col" className="text-left py-2 pr-4">Outlet</th>
                <th scope="col" className="text-left py-2 pr-4">Domain</th>
                <th scope="col" className="text-left py-2 pr-4">Category</th>
                <th scope="col" className="text-left py-2 pr-4">Country</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((s) => (
                <tr key={s.id} className="border-b border-border-subtle hover:bg-bg-sunken">
                  <td className="py-2 pr-4 pl-4 lg:pl-0"><TierGlyph tier={s.trustTier} /></td>
                  <td className="py-2 pr-4 font-medium text-fg-strong">{s.name}</td>
                  <td className="py-2 pr-4 mono text-2xs text-fg-muted">{s.domain}</td>
                  <td className="py-2 pr-4 text-fg-subtle capitalize">{s.category}</td>
                  <td className="py-2 pr-4 mono text-2xs text-fg-muted">{s.country ?? ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {sources !== null && filtered.length === 0 && <p className="text-fg-muted mt-8">No sources match those filters.</p>}
    </div>
  )
}
