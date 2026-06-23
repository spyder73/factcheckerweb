// Masthead-style wordmark + navigation + theme toggle. 64px tall, 1px
// bottom border, restrained. Mobile: hamburger button reveals a sheet.
import { Link, NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { useAuth } from '../hooks/useAuth'
import { ThemeToggle } from './ThemeToggle'
import { cn } from '../utils/cn'

export function TopNav() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)

  const nav = [
    { to: '/check', label: t('nav.check') },
    { to: '/how-it-works', label: t('nav.howItWorks') },
    { to: '/sources', label: t('nav.sources') },
    { to: '/pricing', label: t('nav.pricing') },
  ]
  if (user) nav.splice(1, 0, { to: '/history', label: t('nav.history') })

  return (
    <header className="sticky top-0 z-30 border-b border-border-subtle bg-bg/80 backdrop-blur">
      <nav
        aria-label="Primary"
        className="mx-auto flex h-16 max-w-content items-center justify-between gap-6 px-4 lg:px-8"
      >
        <Link to="/" className="flex items-baseline gap-2 text-fg-strong">
          <span className="text-lg font-bold tracking-tight">Alethea</span>
          <span className="mono text-2xs text-fg-muted" aria-hidden="true">v0.4</span>
        </Link>
        <ul className="hidden items-center gap-1 md:flex">
          {nav.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) => cn(
                  'inline-flex h-8 items-center rounded px-3 text-sm font-medium transition-colors duration-micro',
                  isActive ? 'text-fg-strong bg-bg-sunken' : 'text-fg-muted hover:text-fg hover:bg-bg-sunken',
                )}
              >
                {({ isActive }) => (
                  <span aria-current={isActive ? 'page' : undefined}>{item.label}</span>
                )}
              </NavLink>
            </li>
          ))}
        </ul>
        <div className="flex items-center gap-2">
          <ThemeToggle />
          {user ? (
            <>
              <NavLink to="/settings/keys" className="hidden h-8 items-center px-3 text-sm font-medium text-fg-muted hover:text-fg md:inline-flex">
                {t('nav.settings')}
              </NavLink>
              <button type="button" onClick={() => void logout()} className="hidden h-8 items-center rounded px-3 text-sm font-medium text-fg-muted hover:text-fg md:inline-flex">
                {t('nav.logout')}
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="hidden h-8 items-center px-3 text-sm font-medium text-fg-muted hover:text-fg md:inline-flex">
                {t('nav.login')}
              </Link>
              <Link to="/signup" className="inline-flex h-8 items-center rounded border border-accent bg-accent px-3 text-sm font-medium text-accent-fg transition-colors duration-micro hover:bg-accent-hover">
                {t('nav.signup')}
              </Link>
            </>
          )}
          <button
            type="button"
            className="md:hidden inline-flex h-8 w-8 items-center justify-center rounded border border-border text-fg-muted hover:text-fg"
            aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
            aria-expanded={mobileOpen}
            aria-controls="mobile-nav"
            onClick={() => setMobileOpen((v) => !v)}
          >
            {mobileOpen ? <X size={16} aria-hidden="true" /> : <Menu size={16} aria-hidden="true" />}
          </button>
        </div>
      </nav>
      {mobileOpen && (
        <div id="mobile-nav" className="md:hidden border-t border-border-subtle bg-bg-elevated">
          <ul className="mx-auto max-w-content px-4 py-2">
            {nav.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  onClick={() => setMobileOpen(false)}
                  className={({ isActive }) => cn(
                    'block px-3 py-3 text-base font-medium border-b border-border-subtle last:border-b-0',
                    isActive ? 'text-fg-strong' : 'text-fg-subtle',
                  )}
                >
                  {item.label}
                </NavLink>
              </li>
            ))}
            {user ? (
              <>
                <li>
                  <NavLink to="/settings/keys" onClick={() => setMobileOpen(false)} className="block px-3 py-3 text-base font-medium text-fg-subtle">
                    {t('nav.settings')}
                  </NavLink>
                </li>
                <li>
                  <button type="button" onClick={() => { setMobileOpen(false); void logout() }} className="block w-full text-left px-3 py-3 text-base font-medium text-fg-subtle">
                    {t('nav.logout')}
                  </button>
                </li>
              </>
            ) : (
              <li>
                <Link to="/login" onClick={() => setMobileOpen(false)} className="block px-3 py-3 text-base font-medium text-fg-subtle">
                  {t('nav.login')}
                </Link>
              </li>
            )}
          </ul>
        </div>
      )}
    </header>
  )
}
