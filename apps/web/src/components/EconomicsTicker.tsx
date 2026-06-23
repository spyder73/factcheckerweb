// Thin ticker at the bottom of public pages. Reads /api/economics when
// available; otherwise shows a static "open source" message so the page
// doesn't look broken. Doubled track for seamless scroll.

import { useEffect, useState } from 'react'

interface EconomicsSnapshot {
  spendTodayEUR: number
  checksToday: number
  byokPercent: number
}

export function EconomicsTicker() {
  const [snap, setSnap] = useState<EconomicsSnapshot | null>(null)

  useEffect(() => {
    let cancelled = false
    const base = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '')
    void fetch(`${base}/api/economics`, { credentials: 'include' })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        if (cancelled || !j) return
        setSnap({
          spendTodayEUR: j.spend_today_eur ?? 0,
          checksToday: j.checks_today ?? 0,
          byokPercent: j.byok_percent ?? 0,
        })
      })
      .catch(() => { /* graceful — keep the static fallback */ })
    return () => { cancelled = true }
  }, [])

  const segments: string[] = snap
    ? [
        `TODAY · €${snap.spendTodayEUR.toFixed(2)} spent`,
        `${snap.checksToday.toLocaleString()} checks`,
        `${snap.byokPercent}% BYOK`,
        'funded by donations + Plus subscriptions',
        'open-source · github.com/alethea-app',
      ]
    : [
        'ALETHEA · open-source fact-checking',
        'multi-agent verdicts · skeptical default',
        'every source cited · github.com/alethea-app',
      ]

  // Double the segments so the marquee can scroll seamlessly.
  const track = [...segments, ...segments]

  return (
    <div
      role="status"
      aria-live="off"
      aria-label="Platform economics ticker"
      className="border-t border-border-subtle bg-bg-elevated overflow-hidden"
    >
      <div className="ticker-track animate-ticker-scroll mono text-2xs text-fg-muted py-2">
        {track.map((s, i) => (
          <span key={i} aria-hidden={i >= segments.length}>· {s} </span>
        ))}
      </div>
    </div>
  )
}
