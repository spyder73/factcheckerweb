// AuthProvider holds the session state. Loads from /auth/me on mount; lets
// consumers call login/logout. CSRF token is stored both in client.ts module
// state (in-memory) and a JS-readable cookie (server-set on login), so a
// hard reload + immediate POST still works.

import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import * as authApi from '../api/auth'
import type { UserInfo } from '../types/api'
import { Ctx } from './authContext'
import type { AuthCtx } from './authContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const r = await authApi.me()
      setUser(r?.user ?? null)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const value = useMemo<AuthCtx>(() => ({
    user, loading,
    login: async (req) => {
      const r = await authApi.login(req)
      setUser(r.user)
      return r.user
    },
    signup: async (req) => authApi.signup(req),
    logout: async () => {
      await authApi.logout()
      setUser(null)
    },
    refresh,
  }), [user, loading, refresh])

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
