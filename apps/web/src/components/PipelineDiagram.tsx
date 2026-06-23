// Custom SVG pipeline diagram. Hand-drawn instead of mermaid because mermaid
// renders look utilitarian; this uses the design tokens (accent color, fg-muted
// hairlines) and scales to two viewports:
//   - desktop (>=lg): horizontal flow, 7 nodes left-to-right with fan-out at "Investigate × N"
//   - mobile: vertical flow, same nodes stacked
//
// Variant prop controls verbosity:
//   - "landing":     decorative — node icons + labels only
//   - "explainer":   full — adds hover tooltips with the per-stage description

import { useRef, useState } from 'react'
import { motion, useInView } from 'framer-motion'
import { DURATION, EASING } from '../design/tokens'
import { useReducedMotion } from '../hooks/useReducedMotion'

interface Node {
  id: string
  label: string
  detail: string
}

const NODES: Node[] = [
  { id: 'scrape',     label: 'Scrape',     detail: 'Fetch the post via Instagram service or generic meta-tag scraper. SSRF-guarded — RFC1918 / loopback / link-local / cloud-metadata IPs rejected.' },
  { id: 'media',      label: 'Media',      detail: 'Vision model summarises every attached image into 2-4 sentences of checkable text. Parallel, capped at MAX_MEDIA_ITEMS.' },
  { id: 'extract',    label: 'Extract',    detail: 'Cheap model condenses caption + image analyses into a list of atomic, testable claims. Opinions and jokes filtered out.' },
  { id: 'screen',     label: 'Screen',     detail: 'Per-claim cheap-model pass produces a normalised text + a difficulty hint + uncertainty questions. Drives retrieval queries.' },
  { id: 'retrieve',   label: 'Retrieve',   detail: 'Web search (Brave + Tavily fallback). Pool de-duped by URL, reranked so curated sources surface first. Domain-diversity guard caps confidence if <2 distinct domains.' },
  { id: 'investigate', label: 'Investigate × N', detail: 'N independent investigator agents reason over the SAME shared source pool. Each must cite by pool-ID — they can\'t hallucinate sources. Styles: empiricist / skeptic / historical-context.' },
  { id: 'judge',      label: 'Judge',      detail: 'Strong model synthesizes the N reports into one verdict. Skeptical floor: confidence < 0.65 demotes to "Unverifiable". Dissent shown to the user verbatim.' },
  { id: 'verdict',    label: 'Verdict',    detail: 'Verdict + confidence + summary + dissent + cited sources + intent-interpretation. Content-hashed for non-repudiation, cached for 1h-30d depending on verdict type.' },
]

interface Props {
  variant?: 'landing' | 'explainer'
  className?: string
}

export function PipelineDiagram({ variant = 'landing', className }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const inView = useInView(containerRef, { once: true, amount: 0.3 })
  const reduced = useReducedMotion()
  const animate = inView && !reduced
  const [hoveredId, setHoveredId] = useState<string | null>(null)

  return (
    <div
      ref={containerRef}
      className={['relative w-full', className].filter(Boolean).join(' ')}
      role="img"
      aria-label="Alethea fact-check pipeline: scrape, media, extract, screen, retrieve, investigate with N agents, judge, verdict"
    >
      {/* Desktop: horizontal */}
      <div className="hidden lg:block">
        <HorizontalFlow nodes={NODES} animate={animate} variant={variant} hoveredId={hoveredId} setHoveredId={setHoveredId} />
      </div>
      {/* Mobile / tablet: vertical */}
      <div className="lg:hidden">
        <VerticalFlow nodes={NODES} animate={animate} variant={variant} hoveredId={hoveredId} setHoveredId={setHoveredId} />
      </div>

      {/* Tooltip (explainer variant only) */}
      {variant === 'explainer' && hoveredId && (
        <div
          role="tooltip"
          className="absolute left-1/2 -translate-x-1/2 -bottom-2 translate-y-full w-80 max-w-[90vw] z-10 pointer-events-none"
        >
          <div className="bg-bg-elevated border border-border rounded-md px-4 py-3 shadow-lg">
            <div className="text-eyebrow uppercase text-fg-muted mb-1">
              {NODES.find((n) => n.id === hoveredId)?.label}
            </div>
            <p className="text-sm text-fg">
              {NODES.find((n) => n.id === hoveredId)?.detail}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

// --- Horizontal flow (desktop) ---

function HorizontalFlow({ nodes, animate, variant, hoveredId, setHoveredId }: {
  nodes: Node[]
  animate: boolean
  variant: 'landing' | 'explainer'
  hoveredId: string | null
  setHoveredId: (id: string | null) => void
}) {
  // Geometry: total width 1080 (viewBox), nodes at x = 60, 195, 330, 465, 600, 735, 870, 1020
  // Investigator fan-out at index 5 → 3 sub-nodes branching from a center y to investigate node.
  const xs = [60, 195, 330, 465, 600, 735, 870, 1020]
  const cy = 80
  const r = 30
  const stagger = 0.08
  const interactive = variant === 'explainer'

  return (
    <svg
      viewBox="0 0 1080 200"
      width="100%"
      className="overflow-visible"
      preserveAspectRatio="xMidYMid meet"
    >
      {/* connector lines (drawn first so circles sit on top) */}
      {nodes.slice(0, -1).map((n, i) => {
        const x1 = xs[i] + r
        const x2 = xs[i + 1] - r
        const isInvestigateBranch = n.id === 'screen' // fan into "investigate"
        const isInvestigateRejoin = nodes[i + 1].id === 'judge' && n.id === 'investigate'
        if (isInvestigateBranch) return null // handled by fan-out group
        if (isInvestigateRejoin) return null
        return (
          <motion.line
            key={`line-${i}`}
            x1={x1} y1={cy} x2={x2} y2={cy}
            stroke="rgb(var(--color-border-strong))"
            strokeWidth={1.5}
            strokeDasharray={animate ? 200 : 0}
            strokeDashoffset={animate ? 200 : 0}
            animate={animate ? { strokeDashoffset: 0 } : false}
            transition={{ duration: DURATION.macro, delay: i * stagger + 0.1, ease: EASING.outQuart }}
          />
        )
      })}

      {/* Fan-out from screen → 3 investigator sub-circles → judge */}
      <FanOut animate={animate} startX={xs[4] + r} endX={xs[6] - r} cyCenter={cy} stagger={stagger * 5 + 0.1} />

      {/* nodes */}
      {nodes.map((n, i) => (
        <NodeCircle
          key={n.id}
          node={n}
          x={xs[i]} y={cy} r={r}
          animate={animate}
          delay={i * stagger}
          interactive={interactive}
          hovered={hoveredId === n.id}
          onHover={() => setHoveredId(n.id)}
          onLeave={() => setHoveredId(null)}
          last={i === nodes.length - 1}
        />
      ))}
    </svg>
  )
}

function FanOut({ animate, startX, endX, cyCenter, stagger }: {
  animate: boolean
  startX: number
  endX: number
  cyCenter: number
  stagger: number
}) {
  const yOffsets = [-32, 0, 32]
  const r = 10
  return (
    <g>
      {yOffsets.map((y, i) => {
        const cy = cyCenter + y
        const midX = (startX + endX) / 2
        return (
          <g key={i}>
            {/* line from screen out to mini circle */}
            <motion.path
              d={`M ${startX} ${cyCenter} Q ${midX - 35} ${cyCenter}, ${midX - 10} ${cy}`}
              stroke="rgb(var(--color-accent))" strokeWidth={1.2} fill="none"
              strokeDasharray={animate ? 100 : 0}
              strokeDashoffset={animate ? 100 : 0}
              animate={animate ? { strokeDashoffset: 0 } : false}
              transition={{ duration: DURATION.macro, delay: stagger + i * 0.05, ease: EASING.outQuart }}
            />
            {/* mini investigator circle */}
            <motion.circle
              cx={midX} cy={cy} r={r}
              fill="rgb(var(--color-bg-elevated))"
              stroke="rgb(var(--color-accent))" strokeWidth={1.5}
              initial={animate ? { scale: 0, opacity: 0 } : false}
              animate={animate ? { scale: 1, opacity: 1 } : false}
              transition={{ duration: DURATION.macro, delay: stagger + i * 0.05 + 0.15, ease: EASING.outQuart }}
              style={{ transformOrigin: `${midX}px ${cy}px` }}
            />
            {/* line from mini → judge */}
            <motion.path
              d={`M ${midX + 10} ${cy} Q ${midX + 35} ${cyCenter}, ${endX} ${cyCenter}`}
              stroke="rgb(var(--color-accent))" strokeWidth={1.2} fill="none"
              strokeDasharray={animate ? 100 : 0}
              strokeDashoffset={animate ? 100 : 0}
              animate={animate ? { strokeDashoffset: 0 } : false}
              transition={{ duration: DURATION.macro, delay: stagger + i * 0.05 + 0.25, ease: EASING.outQuart }}
            />
          </g>
        )
      })}
    </g>
  )
}

function NodeCircle({ node, x, y, r, animate, delay, interactive, hovered, onHover, onLeave, last }: {
  node: Node
  x: number; y: number; r: number
  animate: boolean
  delay: number
  interactive: boolean
  hovered: boolean
  onHover: () => void
  onLeave: () => void
  last: boolean
}) {
  return (
    <g
      onMouseEnter={interactive ? onHover : undefined}
      onMouseLeave={interactive ? onLeave : undefined}
      style={{ cursor: interactive ? 'pointer' : 'default' }}
    >
      <motion.circle
        cx={x} cy={y} r={r}
        fill={last ? 'rgb(var(--color-accent-bg))' : 'rgb(var(--color-bg-elevated))'}
        stroke={hovered ? 'rgb(var(--color-accent))' : last ? 'rgb(var(--color-accent))' : 'rgb(var(--color-border-strong))'}
        strokeWidth={hovered ? 2 : 1.5}
        initial={animate ? { scale: 0, opacity: 0 } : false}
        animate={animate ? { scale: 1, opacity: 1 } : false}
        transition={{ duration: DURATION.macro, delay, ease: EASING.outQuart }}
        style={{ transformOrigin: `${x}px ${y}px`, transition: 'stroke 120ms, stroke-width 120ms' }}
      />
      <motion.text
        x={x} y={y + 4}
        textAnchor="middle"
        className="fill-fg-strong"
        style={{ fontSize: 11, fontWeight: 600 }}
        initial={animate ? { opacity: 0 } : false}
        animate={animate ? { opacity: 1 } : false}
        transition={{ duration: DURATION.micro, delay: delay + 0.15, ease: EASING.outQuart }}
      >
        {nodeIcon(node.id)}
      </motion.text>
      <motion.text
        x={x} y={y + r + 18}
        textAnchor="middle"
        className="fill-fg-muted"
        style={{ fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}
        initial={animate ? { opacity: 0, y: y + r + 14 } : false}
        animate={animate ? { opacity: 1, y: y + r + 18 } : false}
        transition={{ duration: DURATION.macro, delay: delay + 0.2, ease: EASING.outQuart }}
      >
        {node.label}
      </motion.text>
    </g>
  )
}

// Two-char text "icons" for each node — keep the design Inter-only, no icon font.
function nodeIcon(id: string): string {
  switch (id) {
    case 'scrape':      return '◷'
    case 'media':       return '▢'
    case 'extract':     return '≡'
    case 'screen':      return '◉'
    case 'retrieve':    return '⌕'
    case 'investigate': return '⋮'
    case 'judge':       return '◆'
    case 'verdict':     return '✓'
    default:            return '·'
  }
}

// --- Vertical flow (mobile) ---

function VerticalFlow({ nodes, animate, variant, hoveredId, setHoveredId }: {
  nodes: Node[]
  animate: boolean
  variant: 'landing' | 'explainer'
  hoveredId: string | null
  setHoveredId: (id: string | null) => void
}) {
  const interactive = variant === 'explainer'
  const reduced = useReducedMotion()

  return (
    <ol className="relative pl-10 space-y-6">
      <div className="absolute left-3.5 top-2 bottom-2 w-0.5 bg-border-strong" aria-hidden="true" />
      {nodes.map((n, i) => {
        const isLast = i === nodes.length - 1
        const isInvestigate = n.id === 'investigate'
        return (
          <motion.li
            key={n.id}
            initial={animate && !reduced ? { opacity: 0, x: -8 } : false}
            animate={animate && !reduced ? { opacity: 1, x: 0 } : false}
            transition={{ duration: DURATION.macro, delay: i * 0.08, ease: EASING.outQuart }}
            className="relative"
            onMouseEnter={interactive ? () => setHoveredId(n.id) : undefined}
            onMouseLeave={interactive ? () => setHoveredId(null) : undefined}
          >
            <span
              aria-hidden="true"
              className={[
                'absolute -left-10 top-1 inline-flex h-7 w-7 items-center justify-center rounded-pill border text-xs font-bold',
                isLast ? 'bg-accent-bg border-accent text-accent' : 'bg-bg-elevated border-border-strong text-fg-strong',
                hoveredId === n.id ? 'ring-2 ring-accent' : '',
              ].join(' ')}
            >
              {nodeIcon(n.id)}
            </span>
            <div>
              <div className="flex items-baseline gap-2">
                <span className="text-eyebrow uppercase text-fg-muted">{n.label}</span>
                {isInvestigate && <span className="mono text-2xs text-accent">× N</span>}
              </div>
              {variant === 'explainer' && (
                <p className="text-sm text-fg-subtle mt-1 prose-measure">{n.detail}</p>
              )}
            </div>
          </motion.li>
        )
      })}
    </ol>
  )
}
