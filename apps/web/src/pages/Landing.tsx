import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowRight } from 'lucide-react'
import { VERDICTS_BY_LEGEND_ORDER } from '../design/tokens'
import { VerdictPill } from '../components/VerdictPill'
import { PipelineDiagram } from '../components/PipelineDiagram'
import { useCountUp } from '../hooks/useCountUp'

export default function Landing() {
  const { t } = useTranslation()

  const m1 = useCountUp(5)
  const m2 = useCountUp(100)
  const m3 = useCountUp(7)
  const m4 = useCountUp(0)
  const metrics = [
    { ref: m1.ref, display: m1.display, suffix: '', l: 'investigators per check' },
    { ref: m2.ref, display: m2.display, suffix: '+', l: 'vetted outlets' },
    { ref: m3.ref, display: m3.display, suffix: '', l: 'verdict categories' },
    { ref: m4.ref, display: m4.display, suffix: '', l: 'data sold (open source)' },
  ]

  return (
    <div>
      {/* Hero */}
      <section className="relative overflow-hidden">
        <div
          className="absolute inset-0 hairline-grid pointer-events-none opacity-40 dark:opacity-30"
          aria-hidden="true"
          style={{ maskImage: 'radial-gradient(ellipse 800px 600px at 50% 30%, black 30%, transparent 80%)', WebkitMaskImage: 'radial-gradient(ellipse 800px 600px at 50% 30%, black 30%, transparent 80%)' }}
        />
        <div className="relative mx-auto max-w-content px-4 lg:px-8 pt-20 pb-16 lg:pt-32">
          <div className="max-w-4xl">
            <p className="inline-flex items-center gap-2 text-eyebrow uppercase text-fg-muted mb-6">
              <span aria-hidden="true" className="inline-block h-1.5 w-1.5 rounded-full bg-accent animate-pulse" />
              OPEN-SOURCE · EVIDENCE-BASED · SKEPTICAL BY DEFAULT
            </p>
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
      <section className="border-t border-b border-border-subtle bg-bg-elevated/40">
        <div className="mx-auto max-w-content px-4 lg:px-8">
          <div className="grid grid-cols-2 lg:grid-cols-4 divide-x divide-border-subtle">
            {metrics.map((m) => (
              <div key={m.l} className="px-6 py-6 first:pl-0 last:pr-0">
                <div className="tabular text-3xl font-bold text-fg-strong">
                  <span ref={m.ref}>{m.display}</span>
                  {m.suffix}
                </div>
                <div className="text-2xs uppercase tracking-wide text-fg-muted mt-1">{m.l}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How it works — pipeline diagram */}
      <section className="mx-auto max-w-content px-4 lg:px-8 py-24">
        <p className="text-eyebrow uppercase text-fg-muted mb-3">HOW IT WORKS</p>
        <h2 className="text-3xl lg:text-4xl font-bold mb-4 max-w-prose">Eight stages, every one visible.</h2>
        <p className="text-fg-subtle prose-measure mb-12">
          A URL goes in, a verdict comes out — but the steps in between are the whole point.
          Nothing is hidden; every claim, every search, every investigator vote, every cited source.
        </p>
        <div className="border border-border-subtle rounded-lg p-6 lg:p-10 bg-bg-elevated/40">
          <PipelineDiagram variant="landing" />
        </div>
        <div className="mt-6 text-center">
          <Link to="/how-it-works" className="text-sm text-accent hover:underline underline-offset-4">
            Read the full methodology →
          </Link>
        </div>
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
