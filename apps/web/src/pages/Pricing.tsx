import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { api } from '../api/client'
import { ApiException } from '../types/api'

const DONATE_URL = import.meta.env.VITE_DONATE_URL || ''

export default function Pricing() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [busy, setBusy] = useState<'monthly' | 'yearly' | 'portal' | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const tiers = [
    { id: 'free', cta: undefined },
    { id: 'byok', cta: undefined },
    { id: 'plus', cta: t('pricing.plus.cta') },
  ] as const

  async function startCheckout(period: 'monthly' | 'yearly') {
    setErr(null)
    if (!user) {
      navigate('/login?next=/pricing')
      return
    }
    setBusy(period)
    try {
      const r = await api<{ url: string }>('/api/me/billing/checkout', {
        method: 'POST',
        body: { period },
      })
      window.location.href = r.url
    } catch (e) {
      if (e instanceof ApiException) {
        setErr(e.code === 'billing_disabled'
          ? 'Billing is not configured on this server yet.'
          : e.message)
      } else {
        setErr('Could not start checkout.')
      }
      setBusy(null)
    }
  }

  async function openPortal() {
    setErr(null)
    setBusy('portal')
    try {
      const r = await api<{ url: string }>('/api/me/billing/portal', {
        method: 'POST',
        body: { returnUrl: window.location.origin + '/pricing' },
      })
      window.location.href = r.url
    } catch (e) {
      if (e instanceof ApiException) setErr(e.message)
      else setErr('Could not open billing portal.')
      setBusy(null)
    }
  }

  const isPlus = user?.plan === 'plus'

  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-4xl lg:text-5xl font-bold mb-2">{t('pricing.title')}</h1>
      <p className="text-fg-muted max-w-2xl mb-12">
        Alethea is open source. You can run it on your own machine for free, bring your own keys,
        or subscribe and we run it for you. Donations keep the lights on.
      </p>

      {err && (
        <div role="alert" className="mb-6 rounded-md border border-verdict-false bg-verdict-false/5 px-4 py-3 text-sm">
          {err}
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-3">
        {tiers.map((tier) => {
          const features = t(`pricing.${tier.id}.features`, { returnObjects: true }) as string[]
          return (
            <article
              key={tier.id}
              className={`border rounded-md p-6 flex flex-col ${tier.id === 'plus' ? 'border-accent shadow-elev1' : 'border-border'}`}
            >
              <h2 className="text-xl font-semibold">{t(`pricing.${tier.id}.name`)}</h2>
              <div className="mt-2 mb-1 text-3xl font-bold text-fg-strong tabular">
                {t(`pricing.${tier.id}.price`)}
              </div>
              {tier.id === 'plus' && (
                <div className="text-sm text-fg-muted mb-4">{t('pricing.plus.yearly')}</div>
              )}
              <ul className="mt-4 space-y-2 text-sm text-fg-subtle flex-1">
                {features.map((f, i) => <li key={i}>· {f}</li>)}
              </ul>

              {tier.id === 'plus' && (
                <div className="mt-6 flex flex-col gap-2">
                  {isPlus ? (
                    <button
                      type="button"
                      onClick={openPortal}
                      disabled={busy !== null}
                      className="h-10 rounded-md bg-accent text-accent-fg font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-micro hover:bg-accent-hover"
                    >
                      {busy === 'portal' ? 'Opening…' : 'Manage subscription'}
                    </button>
                  ) : (
                    <>
                      <button
                        type="button"
                        onClick={() => startCheckout('monthly')}
                        disabled={busy !== null}
                        className="h-10 rounded-md bg-accent text-accent-fg font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-micro hover:bg-accent-hover"
                      >
                        {busy === 'monthly' ? 'Redirecting…' : t('pricing.plus.cta')}
                      </button>
                      <button
                        type="button"
                        onClick={() => startCheckout('yearly')}
                        disabled={busy !== null}
                        className="h-9 rounded-md border border-border text-sm font-medium hover:bg-bg-elevated disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-micro"
                      >
                        {busy === 'yearly' ? 'Redirecting…' : 'Pay yearly (save 2 months)'}
                      </button>
                    </>
                  )}
                </div>
              )}
            </article>
          )
        })}
      </div>

      {DONATE_URL && (
        <div className="mt-12 rounded-md border border-border-subtle bg-bg-elevated p-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <h3 className="font-semibold text-lg">Or just buy us a coffee</h3>
            <p className="text-sm text-fg-muted">
              Donations cover hosting + LLM costs. Every euro is public on our transparency page.
            </p>
          </div>
          <a
            href={DONATE_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="h-10 px-5 inline-flex items-center justify-center rounded-md border border-accent text-accent font-medium hover:bg-accent hover:text-accent-fg transition-colors duration-micro"
          >
            Donate
          </a>
        </div>
      )}
    </div>
  )
}
