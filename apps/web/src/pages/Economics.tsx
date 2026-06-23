export default function Economics() {
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-12">
      <p className="text-eyebrow uppercase text-fg-muted mb-3">RADICAL TRANSPARENCY</p>
      <h1 className="text-4xl lg:text-5xl font-bold mb-6">What this costs us, what you fund</h1>
      <p className="text-lg text-fg-subtle prose-measure mb-12">
        Alethea publishes its compute + search + scrape spend in real time, and how much comes
        from donations vs. Alethea Plus subscriptions. The full live dashboard ships with Phase 5.
        Until then, the ticker at the bottom of every public page shows the rolling snapshot.
      </p>
      <div className="grid gap-6 md:grid-cols-3">
        {['Spend this month', 'Checks this month', 'Funding sources'].map((h) => (
          <div key={h} className="border border-border-subtle rounded-md p-6">
            <p className="text-eyebrow uppercase text-fg-muted">{h}</p>
            <div className="tabular text-3xl font-bold text-fg-strong mt-2">—</div>
            <p className="text-sm text-fg-muted mt-1">live data in Phase 5</p>
          </div>
        ))}
      </div>
    </div>
  )
}
