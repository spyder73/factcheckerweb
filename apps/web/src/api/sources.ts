import { api } from './client'
import type { SourcesResponse } from '../types/api'

export async function listSources(filters?: { category?: string; tier?: string }): Promise<SourcesResponse> {
  return api('/api/sources', { query: filters })
}
