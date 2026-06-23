// Theme toggle. Uses the View Transitions API for a cross-fade swap where
// supported; falls back to an instant attribute change otherwise.
import { Monitor, Moon, Sun } from 'lucide-react'
import { useTheme } from '../hooks/useTheme'
import { useTranslation } from 'react-i18next'

export function ThemeToggle() {
  const { preference, setPreference } = useTheme()
  const { t } = useTranslation()

  const cycle = () => {
    const next = preference === 'system' ? 'light' : preference === 'light' ? 'dark' : 'system'
    const doc = document as Document & { startViewTransition?: (cb: () => void) => unknown }
    if (typeof document !== 'undefined' && typeof doc.startViewTransition === 'function') {
      doc.startViewTransition(() => setPreference(next))
    } else {
      setPreference(next)
    }
  }

  const icon = preference === 'system'
    ? <Monitor size={16} aria-hidden="true" />
    : preference === 'dark' ? <Moon size={16} aria-hidden="true" /> : <Sun size={16} aria-hidden="true" />
  const labels: Record<typeof preference, string> = {
    system: t('common.theme.system'),
    light: t('common.theme.light'),
    dark: t('common.theme.dark'),
  }
  const nextLabel = preference === 'system' ? labels.light : preference === 'light' ? labels.dark : labels.system

  return (
    <button
      type="button"
      onClick={cycle}
      className="inline-flex h-8 w-8 items-center justify-center rounded border border-border text-fg-muted transition-colors duration-micro ease-in-out-quart hover:bg-bg-sunken hover:text-fg"
      title={`${t('common.themeLabel')}: ${labels[preference]} — click to switch to ${nextLabel}`}
      aria-label={`Switch theme to ${nextLabel} (currently ${labels[preference]})`}
    >
      {icon}
    </button>
  )
}
