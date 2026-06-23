import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { forgot } from '../api/auth'

export default function Forgot() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await forgot(email)
      setDone(true)
    } finally {
      setLoading(false)
    }
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md px-4 py-16">
        <h1 className="text-2xl font-bold mb-4">{t('auth.forgot.title')}</h1>
        <p className="text-fg-subtle">{t('auth.forgot.after')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-md px-4 py-16">
      <h1 className="text-3xl font-bold mb-8">{t('auth.forgot.title')}</h1>
      <form onSubmit={onSubmit} className="space-y-4">
        <Input label={t('auth.email')} type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        <Button type="submit" variant="primary" size="lg" loading={loading} className="w-full">{t('auth.forgot.submit')}</Button>
      </form>
    </div>
  )
}
