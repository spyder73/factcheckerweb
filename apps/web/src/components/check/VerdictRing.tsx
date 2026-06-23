// 240px SVG ring (180px on mobile) whose stroke-dashoffset animates from
// circumference → (1 - confidence) * circumference over 600ms. Percentage
// counts up in sync. Respects motion policy: 'none' tier shows the static
// final value with no animation.

import { motion, useMotionValue, useSpring, useTransform } from 'framer-motion'
import { useEffect } from 'react'
import { VERDICTS, DURATION, EASING, resolveMotionPolicy } from '../../design/tokens'
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

  const pct = useMotionValue(policy.allowNarrative ? 0 : Math.round(confidence * 100))
  const pctSpring = useSpring(pct, { duration: policy.allowNarrative ? DURATION.narrative * 1000 : 0, bounce: 0 })
  const pctText = useTransform(pctSpring, (v) => `${Math.round(v)}%`)

  useEffect(() => {
    if (policy.allowNarrative) {
      offset.set(targetOffset)
      pct.set(Math.round(confidence * 100))
    } else {
      offset.set(targetOffset)
      pct.set(Math.round(confidence * 100))
    }
  }, [confidence, policy.allowNarrative, targetOffset, offset, pct])

  return (
    <div
      role="figure"
      aria-label={`Verdict ${meta.label}, confidence ${Math.round(confidence * 100)} percent`}
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
      <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
        <motion.div
          className={`tabular text-5xl font-bold ${meta.textClass}`}
          style={{ opacity: policy.allowNarrative ? undefined : 1 }}
          initial={policy.allowNarrative ? { opacity: 0 } : false}
          animate={policy.allowNarrative ? { opacity: 1 } : false}
          transition={{ duration: DURATION.macro, delay: 0.4, ease: EASING.outQuart }}
        >
          {pctText}
        </motion.div>
        <motion.div
          className={`mt-1 text-eyebrow uppercase font-medium ${meta.textClass}`}
          initial={policy.allowNarrative ? { opacity: 0, y: 4 } : false}
          animate={policy.allowNarrative ? { opacity: 1, y: 0 } : false}
          transition={{ duration: DURATION.macro, delay: 0.45, ease: EASING.outQuart }}
        >
          {meta.label}
        </motion.div>
      </div>
    </div>
  )
}
