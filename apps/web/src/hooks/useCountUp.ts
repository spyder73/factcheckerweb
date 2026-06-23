// Count-up animation hook for trust-strip / metric numerals. Triggers when
// the element enters the viewport (intersection observer). Returns the
// formatted current value. Respects prefers-reduced-motion — jumps to the
// final value if reduced.

import { useEffect, useRef, useState } from 'react'
import { useReducedMotion } from './useReducedMotion'

interface Options {
  duration?: number  // ms
  format?: (n: number) => string
  startOnIntersect?: boolean
}

export function useCountUp(target: number, opts: Options = {}): { ref: React.MutableRefObject<HTMLSpanElement | null>; display: string } {
  const ref = useRef<HTMLSpanElement | null>(null)
  const reduced = useReducedMotion()
  const [value, setValue] = useState(opts.startOnIntersect === false || reduced ? target : 0)
  const fmt = opts.format ?? ((n: number) => Math.round(n).toLocaleString())
  const duration = opts.duration ?? 1100
  const started = useRef(false)

  useEffect(() => {
    if (reduced) {
      setValue(target)
      return
    }
    if (!ref.current || opts.startOnIntersect === false) {
      animate(0, target, duration, setValue)
      started.current = true
      return
    }
    const observer = new IntersectionObserver((entries) => {
      for (const e of entries) {
        if (e.isIntersecting && !started.current) {
          started.current = true
          animate(0, target, duration, setValue)
          observer.disconnect()
        }
      }
    }, { threshold: 0.4 })
    observer.observe(ref.current)
    return () => observer.disconnect()
  }, [target, duration, reduced, opts.startOnIntersect])

  return { ref, display: fmt(value) }
}

function animate(from: number, to: number, durationMs: number, setter: (n: number) => void) {
  const start = performance.now()
  const ease = (t: number) => 1 - Math.pow(1 - t, 4) // ease-out-quart
  function step(now: number) {
    const elapsed = now - start
    const t = Math.min(1, elapsed / durationMs)
    setter(from + (to - from) * ease(t))
    if (t < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}
