import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useAuth } from '../hooks/useAuth'
import { deleteKey, listKeys, setKey } from '../api/byok'
import type { BYOKKey, Provider } from '../types/api'

const PROVIDERS: Provider[] = ['mistral', 'openai', 'anthropic', 'openrouter']

export default function SettingsKeys() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [keys, setKeys] = useState<BYOKKey[] | null>(null)
  const [provider, setProvider] = useState<Provider>('mistral')
  const [secret, setSecret] = useState('')
  const [label, setLabel] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const refresh = async () => {
    try {
      const r = await listKeys()
      setKeys(r.keys)
    } catch {
      setKeys([])
    }
  }
  useEffect(() => { void refresh() }, [])

  if (!user) {
    return <div className="mx-auto max-w-content px-4 py-16 text-fg-muted">Sign in to manage API keys.</div>
  }

  const onAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setErr(null)
    try {
      await setKey(provider, secret, label || undefined)
      setSecret('')
      setLabel('')
      await refresh()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'failed')
    } finally {
      setBusy(false)
    }
  }

  const onDelete = async (p: Provider) => {
    setBusy(true)
    try {
      await deleteKey(p)
      await refresh()
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-3xl font-bold mb-2">{t('settings.keys.title')}</h1>
      <p className="text-fg-subtle prose-measure mb-10">{t('settings.keys.description')}</p>

      <section className="mb-12">
        <h2 className="text-eyebrow uppercase text-fg-muted mb-4">Connected</h2>
        {keys === null && <p className="text-fg-muted">Loading…</p>}
        {keys && keys.length === 0 && <p className="text-fg-muted">{t('settings.keys.noKeys')}</p>}
        {keys && keys.length > 0 && (
          <ul className="border border-border-subtle rounded-md divide-y divide-border-subtle">
            {keys.map((k) => (
              <li key={k.provider} className="flex items-center justify-between px-4 py-3">
                <div>
                  <div className="font-medium text-fg-strong capitalize">{k.provider}</div>
                  {k.label && <div className="text-2xs text-fg-muted">{k.label}</div>}
                  {k.lastUsedAt && <div className="text-2xs text-fg-muted">last used {new Date(k.lastUsedAt).toLocaleString()}</div>}
                </div>
                <Button variant="ghost" size="sm" onClick={() => void onDelete(k.provider)} disabled={busy} iconLeft={<Trash2 size={14} />}>
                  {t('common.delete')}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="text-eyebrow uppercase text-fg-muted mb-4">{t('settings.keys.add')}</h2>
        <form onSubmit={onAdd} className="space-y-4 max-w-xl">
          <label className="block text-sm text-fg-subtle">
            <span className="block mb-2">{t('settings.keys.provider')}</span>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value as Provider)}
              className="w-full bg-bg-elevated border border-border rounded-md px-3 py-2 text-fg"
            >
              {PROVIDERS.map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
          </label>
          <Input label={t('settings.keys.secret')} type="password" value={secret} onChange={(e) => setSecret(e.target.value)} required />
          <Input label={t('settings.keys.label')} type="text" value={label} onChange={(e) => setLabel(e.target.value)} />
          {err && <div role="alert" className="text-sm text-verdict-false">{err}</div>}
          <Button type="submit" variant="primary" loading={busy}>{t('common.save')}</Button>
        </form>
      </section>
    </div>
  )
}
