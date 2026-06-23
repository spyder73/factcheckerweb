import { useTranslation } from 'react-i18next'

export default function Pricing() {
  const { t } = useTranslation()
  const tiers = [
    { id: 'free', cta: undefined },
    { id: 'byok', cta: undefined },
    { id: 'plus', cta: t('pricing.plus.cta') },
  ] as const
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-4xl lg:text-5xl font-bold mb-12">{t('pricing.title')}</h1>
      <div className="grid gap-6 md:grid-cols-3">
        {tiers.map((tier) => {
          const features = t(`pricing.${tier.id}.features`, { returnObjects: true }) as string[]
          return (
            <article key={tier.id} className="border border-border rounded-md p-6 flex flex-col">
              <h2 className="text-xl font-semibold">{t(`pricing.${tier.id}.name`)}</h2>
              <div className="mt-2 mb-1 text-3xl font-bold text-fg-strong tabular">{t(`pricing.${tier.id}.price`)}</div>
              {tier.id === 'plus' && <div className="text-sm text-fg-muted mb-4">{t('pricing.plus.yearly')}</div>}
              <ul className="mt-4 space-y-2 text-sm text-fg-subtle flex-1">
                {features.map((f, i) => <li key={i}>· {f}</li>)}
              </ul>
              {tier.cta && (
                <>
                  <button
                    type="button"
                    disabled
                    aria-disabled="true"
                    aria-describedby="plus-cta-note"
                    className="mt-6 h-10 rounded-md bg-accent text-accent-fg font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-micro"
                  >
                    {tier.cta}
                  </button>
                  <p id="plus-cta-note" className="mt-2 text-2xs text-fg-muted">
                    Billing rolls out in the next release.
                  </p>
                </>
              )}
            </article>
          )
        })}
      </div>
    </div>
  )
}
