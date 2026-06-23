import { Link, useParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { getCheck } from '../api/check'
import type { CheckRow } from '../types/api'
import { ApiException } from '../types/api'
import { useCheck } from '../hooks/useCheck'
import { PipelineLog } from '../components/check/PipelineLog'
import { ResultBlock } from '../components/check/ResultBlock'
import { useHead } from '../utils/head'

export default function CheckDetail() {
  const { id = '' } = useParams()
  const { t } = useTranslation()
  const [row, setRow] = useState<CheckRow | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    void getCheck(id).then(setRow).catch((e: unknown) => {
      setErr(e instanceof ApiException ? e.message : 'failed to load')
    })
  }, [id])

  // If the check is still processing, subscribe to the stream.
  const live = row && (row.status === 'pending' || row.status === 'processing')
  const stream = useCheck(live ? id : null)
  useHead({ noindex: true })

  if (err) {
    return (
      <div className="mx-auto max-w-content px-4 lg:px-8 py-16">
        <h1 className="text-2xl font-semibold mb-2">{t('errors.checkNotFound.title')}</h1>
        <p className="text-fg-subtle mb-6">{t('errors.checkNotFound.body')}</p>
        <Link to="/check" className="text-accent underline">Start a new check →</Link>
      </div>
    )
  }
  if (!row) {
    return <div className="mx-auto max-w-content px-4 lg:px-8 py-16 text-fg-muted">Loading…</div>
  }

  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      {/* Masthead */}
      <div className="mb-8 pb-4 border-b border-border-subtle">
        <p className="mono text-2xs text-fg-muted">
          ALETHEA · check_{row.id.slice(0, 8)} ·{' '}
          {new Date(row.created_at).toISOString().slice(0, 19).replace('T', ' ')} UTC ·{' '}
          {row.completed_at ? `${row.total_cost_micros / 1_000_000 < 0.001 ? '<€0.001' : `€${(row.total_cost_micros / 1_000_000).toFixed(4)}`}` : 'in progress'} ·{' '}
          {row.byok_used ? 'byok' : 'pool'}
        </p>
        <h1 className="text-xl font-semibold mt-2 break-all">{row.input_url}</h1>
      </div>

      {live && <PipelineLog events={stream.events} done={stream.closed} className="mb-10" />}

      {stream.result && <ResultBlock result={stream.result} />}

      {!live && row.status === 'completed' && !stream.result && (
        <div className="rounded-md border border-border bg-bg-elevated p-6 space-y-3">
          <p className="text-fg">This check has already completed.</p>
          <p className="text-sm text-fg-subtle">
            We're still finishing the deep-link viewer that re-plays a finished check from the database.
            Until then, you can <Link to="/check" className="text-accent underline">run the same URL again</Link>{' '}
            to see the live pipeline + verdict.
          </p>
        </div>
      )}
    </div>
  )
}
