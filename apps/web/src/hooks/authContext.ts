import { createContext } from 'react'
import type { UserInfo } from '../types/api'
import type { LoginReq, SignupReq } from '../api/auth'

export interface AuthCtx {
  user: UserInfo | null
  loading: boolean
  login: (req: LoginReq) => Promise<UserInfo>
  signup: (req: SignupReq) => Promise<{ ok: true; message: string }>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

export const Ctx = createContext<AuthCtx | null>(null)
