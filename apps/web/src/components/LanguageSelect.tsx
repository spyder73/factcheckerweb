import { useTranslation } from 'react-i18next'
import { SUPPORTED_LANGS, setLanguage } from '../i18n'
import type { Lang } from '../i18n'

const LABELS: Record<Lang, string> = {
  en: 'English', de: 'Deutsch', es: 'Español', fr: 'Français',
}

export function LanguageSelect() {
  const { i18n, t } = useTranslation()
  return (
    <label className="text-2xs uppercase text-fg-muted">
      <span className="mr-2">{t('common.language')}</span>
      <select
        value={i18n.language}
        onChange={(e) => setLanguage(e.target.value as Lang)}
        className="bg-bg-elevated border border-border rounded px-2 py-1 text-sm text-fg"
      >
        {SUPPORTED_LANGS.map((l) => <option key={l} value={l}>{LABELS[l]}</option>)}
      </select>
    </label>
  )
}
