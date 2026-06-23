// ThemeProvider sets the document's [data-theme] attribute and saves the
// user's preference in localStorage. Must match the bootstrap script in
// index.html — same storage key, same resolution rules.

import { useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { Ctx } from './themeContext'
import type { ThemeCtx, ThemePreference, ResolvedTheme } from './themeContext'

const STORAGE_KEY = 'alethea:theme'

function resolveTheme(pref: ThemePreference): ResolvedTheme {
  if (pref === 'system') {
    if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  }
  return pref
}

function applyTheme(t: ResolvedTheme) {
  if (typeof document === 'undefined') return
  document.documentElement.setAttribute('data-theme', t)
  document.documentElement.style.colorScheme = t
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [preference, setPreferenceState] = useState<ThemePreference>(() => {
    if (typeof window === 'undefined') return 'system'
    const stored = window.localStorage.getItem(STORAGE_KEY) as ThemePreference | null
    return stored ?? 'system'
  })
  const [resolved, setResolved] = useState<ResolvedTheme>(() => resolveTheme(preference))

  useEffect(() => {
    const r = resolveTheme(preference)
    setResolved(r)
    applyTheme(r)
    if (preference === 'system') {
      window.localStorage.removeItem(STORAGE_KEY)
    } else {
      window.localStorage.setItem(STORAGE_KEY, preference)
    }
  }, [preference])

  useEffect(() => {
    if (preference !== 'system' || typeof window === 'undefined') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const listener = () => {
      const r = mq.matches ? 'dark' : 'light'
      setResolved(r)
      applyTheme(r)
    }
    mq.addEventListener('change', listener)
    return () => mq.removeEventListener('change', listener)
  }, [preference])

  const value = useMemo<ThemeCtx>(() => ({
    preference, resolved, setPreference: setPreferenceState,
  }), [preference, resolved])

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
