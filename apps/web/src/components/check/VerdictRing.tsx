// 240px SVG ring (180px on mobile) whose stroke-dashoffset animates from
// circumference → (1 - confidence) * circumference over 600ms. Percentage
// counts up in sync. Respects motion policy: 'none' tier shows the static
// final value with no animation.

import { motion, useMotionValue, useSpring } from 'framer-motion'
import { useEffect } from 'react'
import { VERDICTS, DURATION, EASING, resolveMotionPolicy, confidenceBand, BAND_LABEL } from '../../design/tokens'
import { useReducedMotion } from '../../hooks/useReducedMotion'
import type { Verdict } from '../../types/api'

interface Props {
  verdict: Verdict
  confidence: number      // 0..1
  skepticalFallback?: boolean
  size?: number           // px; default 240, auto-shrunk to fit narrow viewports
  stroke?: number
}

export function VerdictRing({
  verdict, confidence, skepticalFallback = false, size: requestedSize = 240, stroke = 8,
}: Props) {
  // On a 320px viewport with 16px gutters, a 240px ring + adjacent column
  // doesn't fit. Cap at viewport-aware size.
  const size = typeof window !== 'undefined'
    ? Math.min(requestedSize, Math.max(160, window.innerWidth - 64))
    : requestedSize
  const meta = VERDICTS[verdict]
  const reduced = useReducedMotion()
  const policy = resolveMotionPolicy(verdict, skepticalFallback, reduced)

  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const targetOffset = circumference * (1 - confidence)

  const offset = useMotionValue(policy.allowNarrative ? circumference : targetOffset)
  const springOffset = useSpring(offset, {
    duration: policy.allowNarrative ? DURATION.narrative * 1000 : 0,
    bounce: 0,
  })

  useEffect(() => {
    offset.set(targetOffset)
  }, [confidence, targetOffset, offset])

  // M17: bands, not numbers. We never show "0.62" to a non-expert user.
  const band = confidenceBand(confidence)
  const bandLabel = BAND_LABEL[band]

  return (
    <div
      role="figure"
      aria-label={`Verdict ${meta.shortLabel}, ${bandLabel.toLowerCase()}`}
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <circle
          cx={size / 2} cy={size / 2} r={radius}
          fill="none" strokeWidth={stroke}
          className="stroke-border-subtle"
        />
        <motion.circle
          cx={size / 2} cy={size / 2} r={radius}
          fill="none" strokeWidth={stroke}
          className={meta.textClass}
          stroke="currentColor"
          strokeLinecap="round"
          strokeDasharray={circumference}
          style={{ strokeDashoffset: springOffset }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center text-center px-4">
        <motion.div
          className={`text-2xl lg:text-3xl font-bold leading-tight ${meta.textClass}`}
          initial={policy.allowNarrative ? { opacity: 0, y: 8 } : false}
          animate={policy.allowNarrative ? { opacity: 1, y: 0 } : false}
          transition={{ duration: DURATION.macro, delay: 0.4, ease: EASING.outQuart }}
        >
          {meta.shortLabel}
        </motion.div>
        <motion.div
          className="mt-2 text-2xs uppercase tracking-wide font-medium text-fg-muted"
          initial={policy.allowNarrative ? { opacity: 0 } : false}
          animate={policy.allowNarrative ? { opacity: 1 } : false}
          transition={{ duration: DURATION.macro, delay: 0.55, ease: EASING.outQuart }}
        >
          {bandLabel}
        </motion.div>
      </div>
    </div>
  )
}
