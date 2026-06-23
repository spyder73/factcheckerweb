import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { EconomicsTicker } from './EconomicsTicker'
import { LanguageSelect } from './LanguageSelect'

export function Footer({ showTicker = true }: { showTicker?: boolean }) {
  const { t } = useTranslation()
  return (
    <footer className="mt-24 border-t border-border-subtle">
      <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
        <div className="grid gap-8 md:grid-cols-4">
          <div className="space-y-2 md:col-span-2">
            <div className="flex items-baseline gap-2">
              <span className="text-lg font-bold tracking-tight text-fg-strong">Alethea</span>
              <span className="mono text-2xs text-fg-muted">ἀλήθεια</span>
            </div>
            <p className="text-sm text-fg-muted prose-measure">{t('brand.footer')}</p>
          </div>
          <div>
            <h3 className="text-eyebrow uppercase text-fg-muted mb-3">Product</h3>
            <ul className="space-y-2 text-sm">
              <li><Link to="/check" className="text-fg-subtle hover:text-fg">{t('nav.check')}</Link></li>
              <li><Link to="/sources" className="text-fg-subtle hover:text-fg">{t('nav.sources')}</Link></li>
              <li><Link to="/how-it-works" className="text-fg-subtle hover:text-fg">{t('nav.howItWorks')}</Link></li>
              <li><Link to="/pricing" className="text-fg-subtle hover:text-fg">{t('nav.pricing')}</Link></li>
            </ul>
          </div>
          <div>
            <h3 className="text-eyebrow uppercase text-fg-muted mb-3">Open</h3>
            <ul className="space-y-2 text-sm">
              <li><a href="https://github.com/alethea-app" target="_blank" rel="noopener noreferrer" className="text-fg-subtle hover:text-fg">GitHub</a></li>
              <li><Link to="/economics" className="text-fg-subtle hover:text-fg">{t('nav.economics')}</Link></li>
            </ul>
            <div className="mt-4">
              <LanguageSelect />
            </div>
          </div>
        </div>
      </div>
      {showTicker && <EconomicsTicker />}
    </footer>
  )
}
