// Live SSE event timeline. Looks like a terminal log but in our typeface
// rules — JetBrains Mono for stage names + IDs, Inter for human messages.
// Active row has an accent left-border that pulses; settled rows are plain.

import { useEffect, useRef } from 'react'
import { STAGE_LABEL } from '../../design/tokens'
import type { SSEEvent } from '../../types/api'
import { cn } from '../../utils/cn'

interface Props {
  events: SSEEvent[]
  done: boolean
  className?: string
}

export function PipelineLog({ events, done, className }: Props) {
  const listRef = useRef<HTMLOListElement>(null)
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' })
  }, [events.length])

  const lastSeq = events.length > 0 ? events[events.length - 1].seq : -1

  // Live-region pattern: a visually-hidden polite live region announces each
  // new event in text, while the visual <ol> is purely presentational for AT.
  // NVDA/JAWS don't reliably announce list-item insertions with animated entry.
  const latest = events[events.length - 1]

  return (
    <>
      <div
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
        role="status"
      >
        {latest ? `${latest.stage}: ${latest.message || ''}` : ''}
        {done ? ' Pipeline complete.' : ''}
      </div>
      <ol
        ref={listRef}
        aria-hidden="true"
        className={cn(
          'bg-bg-sunken border border-border-subtle rounded-lg p-2 max-h-80 overflow-y-auto',
          className,
        )}
      >
      {events.length === 0 && (
        <li className="text-fg-muted px-3 py-2 text-sm">Waiting for first event…</li>
      )}
      {events.map((e) => {
        const isActive = !done && e.seq === lastSeq
        return (
          <li
            key={e.seq}
            className={cn(
              'flex items-baseline gap-3 px-3 py-1.5 text-sm border-l-2 transition-colors duration-micro animate-sse-row-arrive',
              isActive ? 'border-l-accent animate-sse-row-pulse' : 'border-l-border-subtle',
            )}
          >
            <span className="mono text-2xs w-8 tabular text-fg-muted">{String(e.seq).padStart(2, '0')}</span>
            <span className="mono text-xs text-accent w-32 shrink-0">{e.stage}</span>
            <span className="text-fg flex-1 truncate">{e.message || STAGE_LABEL[e.stage] || ''}</span>
          </li>
        )
      })}
      </ol>
    </>
  )
}
