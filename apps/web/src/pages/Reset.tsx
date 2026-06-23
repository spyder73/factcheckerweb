import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { reset as resetApi } from '../api/auth'
import { ApiException } from '../types/api'

export default function Reset() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [pw, setPw] = useState('')
  const [done, setDone] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    setLoading(true)
    try {
      await resetApi(token, pw)
      setDone(true)
    } catch (e) {
      const msg = e instanceof ApiException ? e.message : 'failed'
      setErr(msg)
    } finally {
      setLoading(false)
    }
  }

  if (!token) {
    return (
      <div className="mx-auto max-w-md px-4 py-16">
        <h1 className="text-2xl font-semibold mb-4">{t('errors.notFound.title')}</h1>
        <p className="text-fg-subtle">The reset link is missing a token. Try requesting another from{' '}
          <Link to="/forgot" className="text-accent">{t('auth.forgot.title')}</Link>.</p>
      </div>
    )
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md px-4 py-16">
        <h1 className="text-2xl font-bold mb-4">Password updated</h1>
        <p className="text-fg-subtle mb-4">All your other sessions have been signed out for safety.</p>
        <Link to="/login" className="text-accent underline">{t('nav.login')} →</Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-md px-4 py-16">
      <h1 className="text-3xl font-bold mb-8">{t('auth.reset.title')}</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <Input label={t('auth.newPassword')} type="password" value={pw} onChange={(e) => setPw(e.target.value)} required minLength={8} />
        {err && <div role="alert" className="text-sm text-verdict-false">{err}</div>}
        <Button type="submit" variant="primary" size="lg" loading={loading} className="w-full">{t('auth.reset.submit')}</Button>
      </form>
    </div>
  )
}
