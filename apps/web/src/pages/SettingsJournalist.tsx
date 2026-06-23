import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useAuth } from '../hooks/useAuth'
import { getMyApplication, submitApplication, withdrawApplication } from '../api/journalist'
import type { JournalistApplication } from '../types/api'

export default function SettingsJournalist() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [app, setApp] = useState<JournalistApplication | null | undefined>(undefined)

  const [form, setForm] = useState({
    fullName: '', outlet: '', outletUrl: '', bylineUrls: '', country: '', beat: '', bio: '',
  })
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const refresh = async () => {
    const r = await getMyApplication()
    setApp(r.application)
  }
  useEffect(() => { void refresh() }, [])

  if (!user) {
    return <div className="mx-auto max-w-content px-4 py-16 text-fg-muted">Sign in to apply.</div>
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    setBusy(true)
    try {
      const urls = form.bylineUrls.split('\n').map((s) => s.trim()).filter(Boolean)
      await submitApplication({
        fullName: form.fullName, outlet: form.outlet, outletUrl: form.outletUrl,
        bylineUrls: urls,
        country: form.country || undefined, beat: form.beat || undefined, bio: form.bio || undefined,
      })
      await refresh()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'failed')
    } finally {
      setBusy(false)
    }
  }

  const onWithdraw = async () => {
    setBusy(true)
    try {
      await withdrawApplication()
      await refresh()
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-3xl font-bold mb-2">{t('settings.journalist.title')}</h1>
      <p className="text-fg-subtle prose-measure mb-10">{t('settings.journalist.description')}</p>

      {app && (
        <section className="mb-10 border border-border-subtle rounded-md p-6">
          <div className="flex items-baseline gap-3 mb-2">
            <h2 className="text-eyebrow uppercase text-fg-muted">Your application</h2>
            <span className="text-sm font-medium text-fg-strong">{t(`settings.journalist.status.${app.status}`)}</span>
          </div>
          <p className="text-sm text-fg-subtle">
            {app.outlet} · {app.outletUrl} · submitted {new Date(app.createdAt).toLocaleDateString()}
          </p>
          {app.status === 'pending' && (
            <Button variant="ghost" size="sm" onClick={() => void onWithdraw()} disabled={busy} className="mt-4">
              {t('settings.journalist.withdraw')}
            </Button>
          )}
        </section>
      )}

      {(!app || app.status !== 'pending') && (
        <form onSubmit={onSubmit} className="space-y-4 max-w-2xl">
          <Input label={t('settings.journalist.fields.fullName')} value={form.fullName} onChange={(e) => setForm({ ...form, fullName: e.target.value })} required />
          <Input label={t('settings.journalist.fields.outlet')} value={form.outlet} onChange={(e) => setForm({ ...form, outlet: e.target.value })} required />
          <Input label={t('settings.journalist.fields.outletUrl')} type="url" value={form.outletUrl} onChange={(e) => setForm({ ...form, outletUrl: e.target.value })} required />
          <label className="block text-sm">
            <span className="block text-2xs uppercase tracking-wide font-medium text-fg-muted mb-2">
              {t('settings.journalist.fields.bylineUrls')}
            </span>
            <textarea
              value={form.bylineUrls}
              onChange={(e) => setForm({ ...form, bylineUrls: e.target.value })}
              rows={4}
              required
              className="w-full bg-bg-elevated border border-border rounded-md px-3 py-2 text-fg font-mono text-sm"
            />
          </label>
          <Input label={t('settings.journalist.fields.country')} value={form.country} onChange={(e) => setForm({ ...form, country: e.target.value })} />
          <Input label={t('settings.journalist.fields.beat')} value={form.beat} onChange={(e) => setForm({ ...form, beat: e.target.value })} />
          <label className="block text-sm">
            <span className="block text-2xs uppercase tracking-wide font-medium text-fg-muted mb-2">
              {t('settings.journalist.fields.bio')}
            </span>
            <textarea
              value={form.bio}
              onChange={(e) => setForm({ ...form, bio: e.target.value })}
              rows={3}
              className="w-full bg-bg-elevated border border-border rounded-md px-3 py-2 text-fg text-sm"
            />
          </label>
          {err && <div role="alert" className="text-sm text-verdict-false">{err}</div>}
          <Button type="submit" variant="primary" loading={busy}>{t('settings.journalist.submit')}</Button>
        </form>
      )}
    </div>
  )
}
