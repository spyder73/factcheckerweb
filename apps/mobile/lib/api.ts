// Thin API client for the mobile app. Mirrors the web client's contract
// (apps/web/src/api/client.ts) so we can share request/response types.
//
// Cookie auth is awkward on RN/iOS so we use a token-in-header model
// instead: after login the server returns {sessionToken, csrfToken}; the
// client stores them in SecureStore and attaches X-Session-Token +
// X-CSRF-Token on every request. The web SPA continues to use the
// HttpOnly cookie path — both flows are valid.

import Constants from 'expo-constants'
import * as SecureStore from 'expo-secure-store'
import { ApiException, type ApiError } from '@alethea/shared-types'

const API_BASE = String(
  Constants.expoConfig?.extra?.apiUrl || 'http://localhost:8080',
).replace(/\/$/, '')

const SESSION_KEY = 'alethea.session'
const CSRF_KEY = 'alethea.csrf'

export async function storeAuth(sessionToken: string, csrfToken: string): Promise<void> {
  await Promise.all([
    SecureStore.setItemAsync(SESSION_KEY, sessionToken),
    SecureStore.setItemAsync(CSRF_KEY, csrfToken),
  ])
}

export async function clearAuth(): Promise<void> {
  await Promise.all([
    SecureStore.deleteItemAsync(SESSION_KEY),
    SecureStore.deleteItemAsync(CSRF_KEY),
  ])
}

export async function getSessionToken(): Promise<string | null> {
  return SecureStore.getItemAsync(SESSION_KEY)
}

export interface FetchOptions {
  method?: 'GET' | 'POST' | 'DELETE' | 'PUT' | 'PATCH'
  body?: unknown
  signal?: AbortSignal
}

export async function api<T = unknown>(path: string, opts: FetchOptions = {}): Promise<T> {
  const method = opts.method ?? 'GET'
  const headers: Record<string, string> = { Accept: 'application/json' }

  const session = await SecureStore.getItemAsync(SESSION_KEY)
  if (session) headers['X-Session-Token'] = session
  if (method !== 'GET') {
    const csrf = await SecureStore.getItemAsync(CSRF_KEY)
    if (csrf) headers['X-CSRF-Token'] = csrf
  }

  let body: string | undefined
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }

  let resp: Response
  try {
    resp = await fetch(API_BASE + path, { method, headers, body, signal: opts.signal })
  } catch (e) {
    throw new ApiException(0, 'network', e instanceof Error ? e.message : 'network error')
  }

  if (resp.status === 204) return undefined as T

  const text = await resp.text()
  let parsed: unknown = null
  if (text.length > 0) {
    try { parsed = JSON.parse(text) } catch {
      if (!resp.ok) throw new ApiException(resp.status, 'non_json', text.slice(0, 200))
    }
  }

  if (!resp.ok) {
    const err = parsed as ApiError | null
    throw new ApiException(
      resp.status,
      err?.error?.code ?? 'http_' + resp.status,
      err?.error?.message ?? `HTTP ${resp.status}`,
    )
  }

  return parsed as T
}
