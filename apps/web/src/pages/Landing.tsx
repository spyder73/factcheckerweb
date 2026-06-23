import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight } from 'lucide-react'
import { VERDICTS_BY_LEGEND_ORDER } from '../design/tokens'
import { VerdictPill } from '../components/VerdictPill'

export default function Landing() {
  const { t } = useTranslation()

  const stages = [
    { key: 'scrape',     label: t('landing.stages.scrape') },
    { key: 'extract',    label: t('landing.stages.extract') },
    { key: 'investigate', label: t('landing.stages.investigate') },
    { key: 'judge',      label: t('landing.stages.judge') },
    { key: 'explain',    label: t('landing.stages.explain') },
  ]

  const metrics = [
    { v: '5', l: 'investigators per check' },
    { v: '100+', l: 'vetted outlets' },
    { v: '7', l: 'verdict categories' },
    { v: '0', l: 'data sold (open source)' },
  ]

  return (
    <div>
      {/* Hero */}
      <section className="relative">
        <div className="absolute inset-0 hairline-grid opacity-30 pointer-events-none hidden lg:block" aria-hidden="true" />
        <div className="relative mx-auto max-w-content px-4 lg:px-8 pt-20 pb-16 lg:pt-32">
          <div className="max-w-4xl">
            <p className="text-eyebrow uppercase text-fg-muted mb-6">OPEN-SOURCE · EVIDENCE-BASED · SKEPTICAL BY DEFAULT</p>
            <h1 className="text-5xl lg:text-7xl font-bold text-fg-strong mb-8 tracking-tight">
              {t('landing.hero.headline')}
            </h1>
            <p className="text-lg lg:text-xl text-fg-subtle max-w-measure mb-10">
              {t('landing.hero.subline')}
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link
                to="/check"
                className="inline-flex items-center gap-2 h-12 px-6 rounded-md bg-accent text-accent-fg font-medium hover:bg-accent-hover transition-colors duration-micro"
              >
                {t('landing.hero.cta')}
                <ArrowRight size={16} aria-hidden="true" />
              </Link>
              <Link
                to="/how-it-works"
                className="inline-flex items-center gap-2 h-12 px-6 rounded-md border border-border text-fg hover:bg-bg-sunken transition-colors duration-micro"
              >
                {t('landing.hero.ctaSecondary')}
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Trust strip */}
      <section className="border-t border-b border-border-subtle">
        <div className="mx-auto max-w-content px-4 lg:px-8">
          <div className="grid grid-cols-2 lg:grid-cols-4 divide-x divide-border-subtle">
            {metrics.map((m) => (
              <div key={m.l} className="px-6 py-4 first:pl-0 last:pr-0">
                <div className="tabular text-2xl font-bold text-fg-strong">{m.v}</div>
                <div className="text-2xs uppercase tracking-wide text-fg-muted mt-1">{m.l}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How it works */}
      <section className="mx-auto max-w-content px-4 lg:px-8 py-20">
        <p className="text-eyebrow uppercase text-fg-muted mb-3">HOW IT WORKS</p>
        <h2 className="text-3xl lg:text-4xl font-bold mb-12 max-w-prose">Five stages, every one visible.</h2>
        <ol className="grid gap-4 lg:grid-cols-5">
          {stages.map((s, i) => (
            <li key={s.key} className="border border-border-subtle rounded-md p-5">
              <span className="mono text-2xs text-fg-muted">{(i + 1).toString().padStart(2, '0')}</span>
              <h3 className="text-base font-semibold mt-2 mb-2">{s.label}</h3>
            </li>
          ))}
        </ol>
      </section>

      {/* Verdict legend */}
      <section className="mx-auto max-w-content px-4 lg:px-8 pb-20">
        <p className="text-eyebrow uppercase text-fg-muted mb-3">VERDICT VOCABULARY</p>
        <h2 className="text-3xl font-bold mb-8 max-w-prose">Seven outcomes. "Unverifiable" is the default if evidence is thin.</h2>
        <ul className="flex flex-wrap gap-2">
          {VERDICTS_BY_LEGEND_ORDER.map((v) => (
            <li key={v.id}>
              <VerdictPill verdict={v.id} />
            </li>
          ))}
        </ul>
      </section>

      {/* Three promises */}
      <section className="mx-auto max-w-content px-4 lg:px-8 pb-20">
        <p className="text-eyebrow uppercase text-fg-muted mb-3">{t('landing.promise.title')}</p>
        <div className="grid gap-8 lg:grid-cols-3">
          <Promise title="Skeptical by default" body={t('landing.promise.skeptical')} />
          <Promise title="Show our work" body={t('landing.promise.transparent')} />
          <Promise title="Open source" body={t('landing.promise.open')} />
        </div>
      </section>
    </div>
  )
}

function Promise({ title, body }: { title: string; body: string }) {
  return (
    <div className="border-t border-border-subtle pt-6">
      <h3 className="text-lg font-semibold text-fg-strong mb-2">{title}</h3>
      <p className="text-fg-subtle">{body}</p>
    </div>
  )
}
