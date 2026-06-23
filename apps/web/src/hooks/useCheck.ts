// useCheck subscribes to /api/check/{id}/stream and surfaces the timeline +
// final result. Auto-reconnect is handled by EventSource itself; we just
// surface state to the component.

import { useEffect, useRef, useState } from 'react'
import { subscribeCheckStream } from '../api/check'
import type { CheckDonePayload, SSEEvent } from '../types/api'

export interface UseCheckState {
  events: SSEEvent[]
  result: CheckDonePayload | null
  error: string | null
  closed: boolean
}

export function useCheck(id: string | null | undefined): UseCheckState {
  const [events, setEvents] = useState<SSEEvent[]>([])
  const [result, setResult] = useState<CheckDonePayload | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [closed, setClosed] = useState(false)
  const unsubRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    if (!id) return
    setEvents([])
    setResult(null)
    setError(null)
    setClosed(false)

    const unsub = subscribeCheckStream(id, {
      onEvent: (ev) => setEvents((prev) => [...prev, ev]),
      onDone: (payload) => {
        setResult(payload)
        setClosed(true)
      },
      onError: (e) => {
        // Non-fatal "connection blip" errors are noisy but expected; only
        // surface terminal ones (those come from stage='error').
        if (e.message.includes('pipeline error') || e.message.includes('SSE connection blip')) {
          // The first message means real failure; the second is just a reconnect.
          if (e.message.includes('pipeline error')) {
            setError(e.message)
            setClosed(true)
          }
        } else {
          setError(e.message)
          setClosed(true)
        }
      },
    })
    unsubRef.current = unsub
    return () => {
      unsub()
      unsubRef.current = null
    }
  }, [id])

  return { events, result, error, closed }
}
