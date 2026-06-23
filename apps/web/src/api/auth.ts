import { api, setCSRF } from './client'
import type { MeResponse } from '../types/api'

export interface SignupReq {
  email: string
  password: string
  captchaToken?: string
}
export interface LoginReq {
  email: string
  password: string
  captchaToken?: string
}

// Signup returns a generic body regardless of duplicate-email status, and
// does NOT set a session cookie. Follow with login() to get authenticated.
export async function signup(req: SignupReq): Promise<{ ok: true; message: string }> {
  return api('/auth/signup', { method: 'POST', body: req })
}

export async function login(req: LoginReq): Promise<MeResponse> {
  const r = await api<MeResponse>('/auth/login', { method: 'POST', body: req })
  setCSRF(r.csrfToken)
  return r
}

export async function logout(): Promise<void> {
  try {
    await api<void>('/auth/logout', { method: 'POST' })
  } finally {
    setCSRF(null)
  }
}

export async function me(): Promise<MeResponse | null> {
  try {
    const r = await api<MeResponse>('/auth/me')
    setCSRF(r.csrfToken)
    return r
  } catch (e: unknown) {
    if (typeof e === 'object' && e && 'status' in e && (e as { status: number }).status === 401) {
      return null
    }
    throw e
  }
}

export async function forgot(email: string, captchaToken?: string): Promise<void> {
  await api<void>('/auth/forgot', { method: 'POST', body: { email, captchaToken } })
}

export async function reset(token: string, newPassword: string): Promise<void> {
  await api<void>('/auth/reset', { method: 'POST', body: { token, newPassword } })
}

export async function verify(token: string): Promise<void> {
  await api<void>('/auth/verify', { method: 'POST', body: { token } })
}
