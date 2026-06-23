import { api } from './client'
import type { JournalistApplication } from '../types/api'

export interface SubmitReq {
  fullName: string
  outlet: string
  outletUrl: string
  bylineUrls: string[]
  country?: string
  beat?: string
  bio?: string
}

export async function submitApplication(req: SubmitReq): Promise<{ id: number; status: string }> {
  return api('/api/me/journalist-application', { method: 'POST', body: req })
}

export async function getMyApplication(): Promise<{ application: JournalistApplication | null }> {
  return api('/api/me/journalist-application')
}

export async function withdrawApplication(): Promise<void> {
  await api<void>('/api/me/journalist-application', { method: 'DELETE' })
}
