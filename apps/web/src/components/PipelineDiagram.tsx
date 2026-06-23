// Custom SVG pipeline diagram. Hand-drawn instead of mermaid because mermaid
// renders look utilitarian; this uses the design tokens (accent color, fg-muted
// hairlines) and scales to two viewports:
//   - desktop (>=lg): horizontal flow, 7 stages left-to-right, with a stack of 3
//     INVESTIGATOR circles in one column so "multiple agents in parallel" reads
//     visually rather than relying on an abstract "× N" notation.
//   - mobile: vertical flow, same stages stacked.
//
// Variant prop controls verbosity:
//   - "landing":     decorative — node icons + labels only
//   - "explainer":   adds hover tooltips with the per-stage description

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
  { id: 'scrape',      label: 'Scrape',        detail: 'Fetch the post via Instagram service or generic meta-tag scraper. SSRF-guarded — RFC1918 / loopback / link-local / cloud-metadata IPs rejected.' },
  { id: 'media',       label: 'Media',         detail: 'Vision model summarises every attached image into 2-4 sentences of checkable text. Parallel, capped at MAX_MEDIA_ITEMS.' },
  { id: 'extract',     label: 'Extract',       detail: 'Cheap model condenses caption + image analyses into a list of atomic, testable claims. Opinions and jokes filtered out.' },
  { id: 'screen',      label: 'Screen',        detail: 'Per-claim cheap-model pass produces a normalised text + a difficulty hint + uncertainty questions. Drives retrieval queries.' },
  { id: 'retrieve',    label: 'Retrieve',      detail: 'Web search (Brave + Tavily fallback). Pool de-duped by URL, reranked so curated sources surface first. Domain-diversity guard caps confidence if <2 distinct domains.' },
  { id: 'investigate', label: 'Investigators', detail: '3 independent agents reason over the SAME shared source pool in parallel. Each must cite by pool-ID — they can\'t hallucinate sources. Styles: empiricist, skeptic, historical-context. Plus tier uses 5.' },
  { id: 'judge',       label: 'Judge',         detail: 'Strong model synthesizes the agent reports into one verdict. Skeptical floor: confidence < 0.65 demotes to "Not enough evidence". Dissent shown to the user verbatim.' },
  { id: 'verdict',     label: 'Verdict',       detail: 'Verdict + confidence + summary + dissent + cited sources + intent-interpretation. Content-hashed for non-repudiation, cached for 1h-30d depending on verdict type.' },
]

// Default agent count shown in the diagram. The pipeline actually scales
// (3 for free/BYOK, 5 for Plus) — this is the visible-to-everyone default.
const DEFAULT_AGENT_COUNT = 3

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
      aria-label={`Alethea fact-check pipeline: scrape, media, extract, screen, retrieve, ${DEFAULT_AGENT_COUNT} investigators in parallel, judge, verdict`}
    >
      {/* Desktop: horizontal */}
      <div className="hidden lg:block">
        <HorizontalFlow nodes={NODES} animate={animate} variant={variant} hoveredId={hoveredId} setHoveredId={setHoveredId} />
      </div>
      {/* Mobile / tablet: vertical */}
      <div className="lg:hidden">
        <VerticalFlow nodes={NODES} animate={animate} variant={variant} hoveredId={hoveredId} setHoveredId={setHoveredId} />
      </div>

      {/* Tooltip (explainer variant only). Lives ABOVE the diagram on hover so
          it never lands on top of the node labels. */}
      {variant === 'explainer' && hoveredId && (
        <div
          role="tooltip"
          className="absolute left-1/2 -translate-x-1/2 -top-2 -translate-y-full w-80 max-w-[90vw] z-10 pointer-events-none hidden lg:block"
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
  // Geometry: 8 columns. The 6th column (investigate) renders a STACK of 3
  // mini-circles at the same x — that's how "multiple in parallel" reads
  // visually, instead of the old wrap-around fan-out which bled into the
  // neighbor labels.
  const xs = [55, 200, 345, 490, 635, 780, 925, 1050]
  const cy = 105
  const r = 30          // main node radius
  const stagger = 0.08
  const interactive = variant === 'explainer'

  // Investigator stack geometry
  const inv = {
    x: xs[5],
    r: 14,                 // smaller — three of them stacked
    ys: [cy - 32, cy, cy + 32], // top / mid / bottom
  }

  return (
    <svg
      viewBox="0 0 1100 195"
      width="100%"
      className="overflow-visible"
      preserveAspectRatio="xMidYMin meet"
    >
      {/* Straight connector lines for the linear stages (skipping the two
          adjacent to the investigator stack — those get fan lines instead). */}
      {nodes.slice(0, -1).map((n, i) => {
        const next = nodes[i + 1]
        if (n.id === 'retrieve' || next.id === 'judge') return null
        const x1 = xs[i] + r
        const x2 = xs[i + 1] - r
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

      {/* Fan lines: retrieve → 3 investigators, and 3 investigators → judge */}
      {inv.ys.map((y, i) => {
        const startX = xs[4] + r            // right edge of retrieve
        const endX = xs[6] - r              // left edge of judge
        const leftStop = inv.x - inv.r
        const rightStart = inv.x + inv.r
        return (
          <g key={`fan-${i}`}>
            <motion.line
              x1={startX} y1={cy} x2={leftStop} y2={y}
              stroke="rgb(var(--color-accent))" strokeWidth={1.2} strokeOpacity={0.7}
              strokeDasharray={animate ? 150 : 0}
              strokeDashoffset={animate ? 150 : 0}
              animate={animate ? { strokeDashoffset: 0 } : false}
              transition={{ duration: DURATION.macro, delay: 4 * stagger + 0.1 + i * 0.04, ease: EASING.outQuart }}
            />
            <motion.line
              x1={rightStart} y1={y} x2={endX} y2={cy}
              stroke="rgb(var(--color-accent))" strokeWidth={1.2} strokeOpacity={0.7}
              strokeDasharray={animate ? 150 : 0}
              strokeDashoffset={animate ? 150 : 0}
              animate={animate ? { strokeDashoffset: 0 } : false}
              transition={{ duration: DURATION.macro, delay: 5 * stagger + 0.15 + i * 0.04, ease: EASING.outQuart }}
            />
          </g>
        )
      })}

      {/* Nodes — render the investigator stack specially. */}
      {nodes.map((n, i) => {
        if (n.id === 'investigate') {
          return (
            <InvestigatorStack
              key={n.id}
              node={n}
              x={inv.x} ys={inv.ys} r={inv.r}
              labelY={cy + r + 22}
              animate={animate}
              delay={i * stagger}
              interactive={interactive}
              hovered={hoveredId === n.id}
              onHover={() => setHoveredId(n.id)}
              onLeave={() => setHoveredId(null)}
            />
          )
        }
        return (
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
        )
      })}
    </svg>
  )
}

function InvestigatorStack({ node, x, ys, r, labelY, animate, delay, interactive, hovered, onHover, onLeave }: {
  node: Node
  x: number
  ys: number[]
  r: number
  labelY: number
  animate: boolean
  delay: number
  interactive: boolean
  hovered: boolean
  onHover: () => void
  onLeave: () => void
}) {
  return (
    <g
      onMouseEnter={interactive ? onHover : undefined}
      onMouseLeave={interactive ? onLeave : undefined}
      style={{ cursor: interactive ? 'pointer' : 'default' }}
    >
      {/* Three stacked agent circles — the visual signal of "multiple in parallel". */}
      {ys.map((y, i) => (
        <motion.circle
          key={i}
          cx={x} cy={y} r={r}
          fill="rgb(var(--color-bg-elevated))"
          stroke={hovered ? 'rgb(var(--color-accent))' : 'rgb(var(--color-accent))'}
          strokeOpacity={hovered ? 1 : 0.85}
          strokeWidth={hovered ? 2 : 1.5}
          initial={animate ? { scale: 0, opacity: 0 } : false}
          animate={animate ? { scale: 1, opacity: 1 } : false}
          transition={{ duration: DURATION.macro, delay: delay + i * 0.05, ease: EASING.outQuart }}
          style={{ transformOrigin: `${x}px ${y}px`, transition: 'stroke 120ms, stroke-width 120ms, stroke-opacity 120ms' }}
        />
      ))}
      {/* Inner dot per circle — same glyph as a single node but echoed thrice. */}
      {ys.map((y, i) => (
        <motion.circle
          key={`dot-${i}`}
          cx={x} cy={y} r={2.5}
          fill="rgb(var(--color-accent))"
          initial={animate ? { opacity: 0 } : false}
          animate={animate ? { opacity: 0.9 } : false}
          transition={{ duration: DURATION.micro, delay: delay + i * 0.05 + 0.15, ease: EASING.outQuart }}
        />
      ))}
      {/* "3" badge at top-right of the stack — concrete count, no abstract × N. */}
      <motion.g
        initial={animate ? { opacity: 0, y: -4 } : false}
        animate={animate ? { opacity: 1, y: 0 } : false}
        transition={{ duration: DURATION.macro, delay: delay + 0.25, ease: EASING.outQuart }}
      >
        <circle cx={x + r + 8} cy={ys[0] - r + 4} r={9}
                fill="rgb(var(--color-accent))" />
        <text x={x + r + 8} y={ys[0] - r + 8}
              textAnchor="middle"
              className="fill-accent-fg"
              style={{ fontSize: 11, fontWeight: 700, fontFamily: 'var(--font-mono, ui-monospace, monospace)' }}>
          {DEFAULT_AGENT_COUNT}
        </text>
      </motion.g>
      {/* Label — single line, no "× N" abstraction. */}
      <motion.text
        x={x} y={labelY}
        textAnchor="middle"
        className="fill-fg-muted"
        style={{ fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}
        initial={animate ? { opacity: 0 } : false}
        animate={animate ? { opacity: 1 } : false}
        transition={{ duration: DURATION.macro, delay: delay + 0.2, ease: EASING.outQuart }}
      >
        {node.label}
      </motion.text>
      <motion.text
        x={x} y={labelY + 13}
        textAnchor="middle"
        className="fill-fg-subtle"
        style={{ fontSize: 9, letterSpacing: '0.02em', fontFamily: 'var(--font-mono, ui-monospace, monospace)' }}
        initial={animate ? { opacity: 0 } : false}
        animate={animate ? { opacity: 1 } : false}
        transition={{ duration: DURATION.macro, delay: delay + 0.25, ease: EASING.outQuart }}
      >
        {DEFAULT_AGENT_COUNT} in parallel
      </motion.text>
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
        style={{ fontSize: 13, fontWeight: 600 }}
        initial={animate ? { opacity: 0 } : false}
        animate={animate ? { opacity: 1 } : false}
        transition={{ duration: DURATION.micro, delay: delay + 0.15, ease: EASING.outQuart }}
      >
        {nodeIcon(node.id)}
      </motion.text>
      <motion.text
        x={x} y={y + r + 22}
        textAnchor="middle"
        className="fill-fg-muted"
        style={{ fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}
        initial={animate ? { opacity: 0 } : false}
        animate={animate ? { opacity: 1 } : false}
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
                {isInvestigate && (
                  <span className="mono text-2xs text-accent">
                    {DEFAULT_AGENT_COUNT} in parallel
                  </span>
                )}
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
