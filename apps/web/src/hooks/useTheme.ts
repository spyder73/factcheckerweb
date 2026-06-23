import { useContext } from 'react'
import { Ctx } from './themeContext'
import type { ThemeCtx } from './themeContext'

export function useTheme(): ThemeCtx {
  const c = useContext(Ctx)
  if (!c) throw new Error('useTheme must be used inside <ThemeProvider>')
  return c
}
