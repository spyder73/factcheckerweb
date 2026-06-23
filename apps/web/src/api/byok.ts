import { api } from './client'
import type { BYOKKey, Provider } from '../types/api'

export async function listKeys(): Promise<{ keys: BYOKKey[] }> {
  return api('/api/me/keys')
}

export async function setKey(provider: Provider, key: string, label?: string): Promise<void> {
  await api<void>('/api/me/keys', { method: 'POST', body: { provider, key, label } })
}

export async function deleteKey(provider: Provider): Promise<void> {
  await api<void>('/api/me/keys', { method: 'DELETE', query: { provider } })
}
