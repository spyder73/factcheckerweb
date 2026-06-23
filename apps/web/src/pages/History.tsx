import { useAuth } from '../hooks/useAuth'
import { Link } from 'react-router-dom'

export default function History() {
  const { user, loading } = useAuth()
  if (loading) {
    return <div className="mx-auto max-w-content px-4 py-16 text-fg-muted">Loading…</div>
  }
  if (!user) {
    return (
      <div className="mx-auto max-w-content px-4 py-16">
        <p className="text-fg-subtle mb-4">Sign in to see your check history.</p>
        <Link to="/login" className="text-accent underline">Sign in →</Link>
      </div>
    )
  }
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <h1 className="text-3xl font-bold mb-4">Your checks</h1>
      <p className="text-fg-subtle mb-8 prose-measure">
        You haven't run any checks yet. Once you do, they'll show up here so you can revisit
        the verdicts and the evidence behind them.
      </p>
      <div className="border border-border-subtle rounded-md p-12 text-center">
        <p className="text-fg-muted mb-4">No checks yet.</p>
        <Link
          to="/check"
          className="inline-flex h-10 items-center rounded-md bg-accent px-4 text-sm font-medium text-accent-fg hover:bg-accent-hover transition-colors duration-micro"
        >
          Start your first check
        </Link>
      </div>
    </div>
  )
}
