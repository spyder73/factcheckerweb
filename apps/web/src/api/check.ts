import { api } from './client'
import type { CheckRow, SSEEvent, SSEStage, CheckDonePayload } from '../types/api'

export interface StartCheckReq {
  url: string
  caption?: string
}

export async function startCheck(req: StartCheckReq): Promise<{ id: string; status: string }> {
  return api('/api/check', { method: 'POST', body: req })
}

export async function getCheck(id: string): Promise<CheckRow> {
  return api(`/api/check/${id}`)
}

// Subscribe to the pipeline's SSE stream. Returns an unsubscribe function.
// onEvent fires on every event; onDone fires once when the pipeline signals
// completion with the final result payload; onError fires on transport error
// or stage='error'. The connection auto-closes after onDone/onError.
export function subscribeCheckStream(
  id: string,
  callbacks: {
    onEvent?: (ev: SSEEvent) => void
    onDone?: (payload: CheckDonePayload) => void
    onError?: (err: Error) => void
  },
  lastEventId?: number,
): () => void {
  const base = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')
  const url = new URL(`${base}/api/check/${id}/stream`)
  if (lastEventId !== undefined) {
    url.searchParams.set('last_event_id', String(lastEventId))
  }

  // EventSource doesn't carry cookies cross-origin by default; need
  // withCredentials. Browser will then send our session cookie if any.
  const es = new EventSource(url.toString(), { withCredentials: true })

  const handle = (raw: MessageEvent) => {
    let ev: SSEEvent
    try {
      ev = JSON.parse(raw.data)
    } catch {
      callbacks.onError?.(new Error('malformed SSE event'))
      return
    }
    callbacks.onEvent?.(ev)
    if (ev.stage === 'done') {
      const payload = ev.payload as unknown as CheckDonePayload
      callbacks.onDone?.(payload)
      es.close()
    } else if (ev.stage === 'error') {
      callbacks.onError?.(new Error(ev.message || 'pipeline error'))
      es.close()
    }
  }

  // Server emits per-stage event names; we listen on every known stage so
  // EventSource fires the matching addEventListener, AND also listen on
  // the generic 'message' as a fallback.
  const stages: SSEStage[] = [
    'init', 'resolve', 'media', 'extract', 'claims', 'screen', 'retrieval',
    'retrieval_failed', 'pool', 'fanout', 'investigator', 'judge', 'cache_hit',
    'claim_done', 'byok_fallback', 'done', 'error',
  ]
  stages.forEach((s) => es.addEventListener(s, handle as EventListener))
  es.addEventListener('end', () => es.close())
  es.onmessage = handle
  es.onerror = () => {
    // EventSource auto-reconnects on transient errors. We surface a non-fatal
    // notification but DON'T close — let it reconnect.
    callbacks.onError?.(new Error('SSE connection blip; retrying'))
  }

  return () => es.close()
}
