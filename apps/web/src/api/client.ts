// Low-level fetch wrapper. Handles:
//   - base URL resolution (VITE_API_URL env, defaults to localhost:8080)
//   - cookie credentials (session cookie is HttpOnly; cookies still flow)
//   - X-CSRF-Token header attached automatically for mutating methods
//   - JSON encoding + error-envelope unwrapping into ApiException
//   - typed JSON response or "no content" 204 handling
//
// CSRF token source: useAuth() stores it in memory after login (the server
// returns it in the /auth/login + /auth/me response bodies); the JS-readable
// alethea_csrf cookie is a fallback for hard reloads where memory is empty.

import { ApiError, ApiException } from '../types/api'

const API_BASE = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')

let csrfToken: string | null = null

export function setCSRF(token: string | null): void {
  csrfToken = token
}

export function getCSRF(): string | null {
  if (csrfToken) return csrfToken
  // Fallback: read the JS-readable alethea_csrf cookie set on login.
  const match = document.cookie.match(/(?:^|;\s*)alethea_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : null
}

export interface FetchOptions {
  method?: 'GET' | 'POST' | 'DELETE' | 'PUT' | 'PATCH'
  body?: unknown
  signal?: AbortSignal
  query?: Record<string, string | number | undefined>
}

export async function api<T = unknown>(path: string, opts: FetchOptions = {}): Promise<T> {
  const method = opts.method ?? 'GET'
  const url = new URL(API_BASE + path)
  if (opts.query) {
    for (const [k, v] of Object.entries(opts.query)) {
      if (v !== undefined && v !== null && v !== '') {
        url.searchParams.set(k, String(v))
      }
    }
  }

  const headers: Record<string, string> = { Accept: 'application/json' }
  let body: BodyInit | undefined
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }
  if (method !== 'GET') {
    const token = getCSRF()
    if (token) headers['X-CSRF-Token'] = token
  }

  let resp: Response
  try {
    resp = await fetch(url.toString(), {
      method,
      headers,
      body,
      credentials: 'include',
      signal: opts.signal,
    })
  } catch (e) {
    throw new ApiException(0, 'network', e instanceof Error ? e.message : 'network error')
  }

  if (resp.status === 204) {
    return undefined as T
  }

  const text = await resp.text()
  let parsed: unknown = null
  if (text.length > 0) {
    try {
      parsed = JSON.parse(text)
    } catch {
      if (!resp.ok) {
        throw new ApiException(resp.status, 'non_json', text.slice(0, 200))
      }
    }
  }

  if (!resp.ok) {
    const err = parsed as ApiError | null
    const code = err?.error?.code ?? 'http_' + resp.status
    const msg = err?.error?.message ?? `HTTP ${resp.status}`
    throw new ApiException(resp.status, code, msg)
  }

  return parsed as T
}
