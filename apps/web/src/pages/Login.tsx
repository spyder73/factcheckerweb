import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useAuth } from '../hooks/useAuth'
import { ApiException } from '../types/api'

export default function Login() {
  const { t } = useTranslation()
  const { login } = useAuth()
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    setLoading(true)
    try {
      await login({ email, password })
      nav('/check')
    } catch (e) {
      const msg = e instanceof ApiException && e.code === 'invalid_credentials'
        ? t('auth.errors.invalid_credentials')
        : e instanceof Error ? e.message : 'failed'
      setErr(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-md px-4 py-16">
      <h1 className="text-3xl font-bold mb-8">{t('auth.login.title')}</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <Input label={t('auth.email')} type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        <Input label={t('auth.password')} type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        {err && <div role="alert" className="text-sm text-verdict-false">{err}</div>}
        <Button type="submit" variant="primary" size="lg" loading={loading} className="w-full">{t('auth.login.submit')}</Button>
      </form>
      <div className="mt-6 text-sm text-fg-muted flex justify-between">
        <Link to="/forgot" className="hover:text-fg">{t('auth.login.forgot')}</Link>
        <Link to="/signup" className="hover:text-fg">{t('nav.signup')}</Link>
      </div>
    </div>
  )
}
