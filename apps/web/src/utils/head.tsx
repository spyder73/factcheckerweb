// Tiny imperative head manager. Single-source <head> manipulation so verdict
// pages can drop noindex meta + ClaimReview JSON-LD without pulling in react-helmet.
// All updates are reverted on unmount.

import { useEffect } from 'react'

interface HeadOpts {
  /** Phase A M11: verdict pages should be noindex,nofollow by default — the
   *  worst case is a screenshot of "WELL-SUPPORTED" laundering disinfo via
   *  Google search results. Opt-in transparency-log publication later. */
  noindex?: boolean
  /** Phase A M20: ClaimReview JSON-LD is table stakes for fact-check-ecosystem
   *  credibility (Google News, Bing, IFCN). Cheap to emit. */
  claimReview?: ClaimReviewSchema
  /** Override the document title for the duration of the page. */
  title?: string
}

export interface ClaimReviewSchema {
  claimReviewed: string
  url: string
  reviewer: string
  reviewBody: string
  // 1..5; lower = "less true"
  ratingValue: number
  bestRating?: number
  worstRating?: number
  alternateName: string  // human label e.g. "Well-supported", "Couldn't verify"
}

export function useHead(opts: HeadOpts): void {
  useEffect(() => {
    const cleanups: Array<() => void> = []

    if (opts.title) {
      const original = document.title
      document.title = opts.title
      cleanups.push(() => { document.title = original })
    }

    if (opts.noindex) {
      const tag = document.createElement('meta')
      tag.name = 'robots'
      tag.content = 'noindex,nofollow'
      document.head.appendChild(tag)
      cleanups.push(() => { document.head.removeChild(tag) })
    }

    if (opts.claimReview) {
      const script = document.createElement('script')
      script.type = 'application/ld+json'
      script.textContent = JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'ClaimReview',
        datePublished: new Date().toISOString().slice(0, 10),
        url: opts.claimReview.url,
        claimReviewed: opts.claimReview.claimReviewed,
        itemReviewed: { '@type': 'Claim', appearance: opts.claimReview.url },
        author: { '@type': 'Organization', name: 'Alethea', url: 'https://alethea.app' },
        reviewBody: opts.claimReview.reviewBody,
        reviewRating: {
          '@type': 'Rating',
          ratingValue: opts.claimReview.ratingValue,
          bestRating: opts.claimReview.bestRating ?? 5,
          worstRating: opts.claimReview.worstRating ?? 1,
          alternateName: opts.claimReview.alternateName,
        },
      })
      document.head.appendChild(script)
      cleanups.push(() => { document.head.removeChild(script) })
    }

    return () => { cleanups.forEach((fn) => fn()) }
  }, [
    opts.title,
    opts.noindex,
    opts.claimReview?.claimReviewed,
    opts.claimReview?.url,
    opts.claimReview?.ratingValue,
  ])
}

// Map our verdict enum to a ClaimReview ratingValue (1=worst, 5=best).
// Schema.org ClaimReview uses an arbitrary scale; this is the conventional
// fact-check mapping (matches Snopes/PolitiFact translations).
export function verdictToRating(verdict: string): { value: number; label: string } {
  switch (verdict) {
    case 'verified':       return { value: 5, label: 'Well-supported' }
    case 'partially_true': return { value: 3, label: 'Partly true' }
    case 'misleading':     return { value: 2, label: 'Misleading' }
    case 'false':          return { value: 1, label: 'Contradicted by sources' }
    case 'unverifiable':   return { value: 0, label: "Couldn't verify" }
    case 'satire':         return { value: 0, label: 'Likely satire' }
    default:               return { value: 0, label: 'No checkable claim' }
  }
}
