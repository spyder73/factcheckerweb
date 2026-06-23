// Heart of the product. Two states:
//   1. INPUT — URL field + sample chips
//   2. LIVE  — PipelineLog streaming + final ResultBlock when done
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { PipelineLog } from '../components/check/PipelineLog'
import { ResultBlock } from '../components/check/ResultBlock'
import { useCheck } from '../hooks/useCheck'
import { startCheck } from '../api/check'
import { ApiException } from '../types/api'
import { useHead, verdictToRating } from '../utils/head'

// Sample URLs disabled until we have permanent, real demo permalinks
// (example.com fails the SSRF guard and confuses users with a broken demo).
// Re-enable once we have 3-4 known-good URLs to a real Instagram/X post + a
// news article + a satire post that we promise to keep alive.
const SAMPLES: { label: string; url: string }[] = []

export default function Check() {
  const { t } = useTranslation()
  const [url, setUrl] = useState('')
  const [caption, setCaption] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [checkId, setCheckId] = useState<string | null>(null)
  const [submitErr, setSubmitErr] = useState<string | null>(null)

  const stream = useCheck(checkId)
  // M23 (Phase A): pipeline timeline is OFF by default. Three reviewers
  // independently flagged the live SSE log as "trust theatre" that
  // accidentally cues authority. Power users can opt in.
  const [showTimeline, setShowTimeline] = useState(false)

  // M11 + M20: noindex on every verdict-rendering page so screenshotted
  // badges can't be SEO-laundered, plus ClaimReview JSON-LD when a verdict
  // is available so the fact-check ecosystem can discover us (opt-in to
  // indexing comes later via a per-check publish flag).
  const firstClaim = stream.result?.claims[0]
  useHead({
    noindex: true,
    claimReview: stream.result && firstClaim
      ? {
          claimReviewed: firstClaim.claim.raw_text,
          url: typeof window !== 'undefined' ? window.location.href : '',
          reviewer: 'Alethea',
          reviewBody: firstClaim.final.summary,
          ratingValue: verdictToRating(firstClaim.final.verdict).value,
          alternateName: verdictToRating(firstClaim.final.verdict).label,
        }
      : undefined,
  })

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!url.trim()) return
    setSubmitErr(null)
    setSubmitting(true)
    try {
      const r = await startCheck({ url: url.trim(), caption: caption.trim() || undefined })
      setCheckId(r.id)
    } catch (e) {
      const msg = e instanceof ApiException ? e.message : 'Network error — try again'
      setSubmitErr(msg)
    } finally {
      setSubmitting(false)
    }
  }

  const reset = () => {
    setCheckId(null)
    setUrl('')
    setCaption('')
    setSubmitErr(null)
  }

  // INPUT state
  if (!checkId) {
    return (
      <div className="mx-auto max-w-content px-4 lg:px-8 py-16 lg:py-24">
        <p className="text-eyebrow uppercase text-fg-muted mb-3">CHECK A POST</p>
        <h1 className="text-4xl lg:text-5xl font-bold text-fg-strong mb-4 max-w-3xl">
          Paste a link to anything — Instagram, X, TikTok, news.
        </h1>
        <p className="text-lg text-fg-subtle max-w-measure mb-10">
          You'll watch the pipeline work in real time: scrape → extract claims → search → judge.
        </p>
        <form onSubmit={onSubmit} className="space-y-4 max-w-2xl">
          <Input
            label={t('check.input.label')}
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder={t('check.input.placeholder')}
            required
            autoFocus
          />
          <Input
            label={t('check.input.captionLabel')}
            type="text"
            value={caption}
            onChange={(e) => setCaption(e.target.value)}
            description={t('check.input.captionPlaceholder')}
          />
          {submitErr && (
            <div role="alert" className="text-sm text-verdict-false">{submitErr}</div>
          )}
          <Button type="submit" variant="primary" size="lg" loading={submitting} disabled={!url.trim()}>
            {t('check.input.submit')}
          </Button>
        </form>

        {SAMPLES.length > 0 && (
          <div className="mt-12 border-t border-border-subtle pt-6">
            <p className="text-eyebrow uppercase text-fg-muted mb-3">TRY A SAMPLE</p>
            <div className="flex flex-wrap gap-2">
              {SAMPLES.map((s) => (
                <button
                  key={s.label}
                  type="button"
                  onClick={() => setUrl(s.url)}
                  aria-label={`Try sample: ${s.label}`}
                  className="rounded-pill border border-border-subtle bg-bg-elevated px-3 py-1.5 text-sm text-fg-subtle hover:border-border hover:text-fg transition-colors duration-micro"
                >
                  {s.label}
                </button>
              ))}
            </div>
          </div>
        )}

        <div className="mt-10 border-t border-border-subtle pt-6 max-w-2xl">
          <p className="text-sm text-fg-muted">
            What happens next: we fetch the page, summarise any images, extract the testable
            claims, and ask N independent investigators to find evidence. You'll watch each
            stage stream into the log above before the final verdict appears.
          </p>
        </div>
      </div>
    )
  }

  // LIVE state
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <div className="flex items-baseline justify-between mb-6">
        <div>
          <p className="text-eyebrow uppercase text-fg-muted">CHECK</p>
          <h1 className="mono text-lg text-fg-strong">check_{checkId.slice(0, 8)}</h1>
        </div>
        <button onClick={reset} className="text-sm text-fg-muted hover:text-fg underline-offset-4 hover:underline">
          New check
        </button>
      </div>

      {/* Quiet-by-default progress: a single line + progress ring. Power users
          can expand to see every SSE event. */}
      {!stream.closed && !stream.result && (
        <div className="mb-10 flex items-center justify-between rounded-lg border border-border-subtle bg-bg-elevated px-5 py-4">
          <div className="flex items-center gap-3">
            <span className="inline-block h-2 w-2 rounded-full bg-accent animate-pulse" aria-hidden="true" />
            <span className="text-fg">{stream.events.length > 0 ? `${stream.events[stream.events.length - 1].message || stream.events[stream.events.length - 1].stage}…` : 'Starting…'}</span>
          </div>
          <button
            type="button"
            onClick={() => setShowTimeline((v) => !v)}
            className="text-sm text-fg-muted hover:text-fg underline-offset-4 hover:underline"
            aria-expanded={showTimeline}
          >
            {showTimeline ? 'Hide details' : 'Show details'}
          </button>
        </div>
      )}
      {showTimeline && <PipelineLog events={stream.events} done={stream.closed} className="mb-10" />}

      {stream.error && !stream.result && (
        <div role="alert" className="rounded-md border border-verdict-false bg-verdict-false-bg p-4 text-fg space-y-2">
          <p className="font-medium">The pipeline stopped.</p>
          <p className="text-sm text-fg-subtle">{stream.error}</p>
          <Button type="button" variant="ghost" size="sm" onClick={reset}>
            Start a new check
          </Button>
        </div>
      )}

      {stream.result && <ResultBlock result={stream.result} />}
    </div>
  )
}
