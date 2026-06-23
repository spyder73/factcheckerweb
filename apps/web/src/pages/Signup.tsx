import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useAuth } from '../hooks/useAuth'

export default function Signup() {
  const { t } = useTranslation()
  const { signup, login } = useAuth()
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [done, setDone] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    setLoading(true)
    try {
      await signup({ email, password })
      // Signup deliberately doesn't set a session (anti-enumeration). The
      // best UX is to immediately try login with the same credentials — that
      // succeeds for a new account and fails harmlessly for a duplicate
      // (the user can then click "Sign in" or "Forgot").
      try {
        await login({ email, password })
        nav('/check')
        return
      } catch {
        // Login failed → most likely a duplicate signup. Show the "check
        // your inbox" page; user can sign in manually.
        setDone(true)
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'failed')
    } finally {
      setLoading(false)
    }
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md px-4 py-16">
        <h1 className="text-2xl font-bold mb-4">{t('auth.signup.afterTitle')}</h1>
        <p className="text-fg-subtle mb-6">{t('auth.signup.afterBody')}</p>
        <Link to="/login" className="text-accent underline">{t('nav.login')} →</Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-md px-4 py-16">
      <h1 className="text-3xl font-bold mb-8">{t('auth.signup.title')}</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <Input label={t('auth.email')} type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        <Input label={t('auth.password')} type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} description="Minimum 8 characters." />
        {err && <div role="alert" className="text-sm text-verdict-false">{err}</div>}
        <Button type="submit" variant="primary" size="lg" loading={loading} className="w-full">{t('auth.signup.submit')}</Button>
      </form>
      <p className="mt-6 text-sm text-fg-muted text-center">
        Already have an account? <Link to="/login" className="text-accent">{t('nav.login')}</Link>
      </p>
    </div>
  )
}
