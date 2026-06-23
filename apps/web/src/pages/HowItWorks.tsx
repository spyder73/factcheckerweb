import { useTranslation } from 'react-i18next'
import { VERDICTS_BY_LEGEND_ORDER, TIERS } from '../design/tokens'
import { VerdictPill } from '../components/VerdictPill'
import { PipelineDiagram } from '../components/PipelineDiagram'

export default function HowItWorks() {
  const { t } = useTranslation()
  const stages = ['scrape', 'extract', 'investigate', 'judge', 'explain'] as const
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <p className="text-eyebrow uppercase text-fg-muted mb-3">METHODOLOGY</p>
      <h1 className="text-4xl lg:text-5xl font-bold mb-8">{t('howItWorks.title')}</h1>
      <p className="text-lg text-fg-subtle prose-measure mb-12">{t('howItWorks.pipeline')}</p>

      <section aria-labelledby="pipeline-diagram-heading" className="border border-border-subtle rounded-lg px-6 pt-10 pb-14 lg:px-10 lg:pt-14 lg:pb-16 bg-bg-elevated/40 mb-16">
        <div className="flex items-baseline justify-between mb-8">
          <h2 id="pipeline-diagram-heading" className="text-eyebrow uppercase text-fg-muted">The pipeline</h2>
          <span className="hidden lg:inline text-2xs text-fg-subtle">Hover any node for details</span>
          <span className="lg:hidden text-2xs text-fg-subtle">Each stage explained below</span>
        </div>
        <PipelineDiagram variant="explainer" />
      </section>

      <section className="space-y-10 mb-16">
        {stages.map((s, i) => (
          <article key={s} className="border-t border-border-subtle pt-6">
            <div className="flex items-baseline gap-3 mb-3">
              <span className="mono text-2xs text-fg-muted">{(i + 1).toString().padStart(2, '0')}</span>
              <h2 className="text-xl font-semibold capitalize">{s}</h2>
            </div>
            <p className="prose-measure text-fg">{t(`howItWorks.stages.${s}`)}</p>
          </article>
        ))}
      </section>

      <section className="border-t border-border-subtle pt-10 mb-16">
        <h2 className="text-2xl font-semibold mb-6">{t('howItWorks.verdicts.title')}</h2>
        <ul className="space-y-3">
          {VERDICTS_BY_LEGEND_ORDER.map((v) => (
            <li key={v.id} className="flex items-start gap-4">
              <VerdictPill verdict={v.id} />
              <p className="text-fg-subtle prose-measure">{t(`howItWorks.verdicts.${v.id}`)}</p>
            </li>
          ))}
        </ul>
      </section>

      <section className="border-t border-border-subtle pt-10">
        <h2 className="text-2xl font-semibold mb-6">Trust tiers</h2>
        <ul className="space-y-3">
          {Object.values(TIERS).map((t) => (
            <li key={t.tier} className="flex items-start gap-4">
              <span className="mono w-24 text-fg-muted">{t.glyph}</span>
              <div>
                <div className="font-medium">{t.label}</div>
                <div className="text-sm text-fg-subtle">{t.description}</div>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
