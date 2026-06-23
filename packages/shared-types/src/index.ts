// Canonical API types for Alethea. Both apps/web and apps/mobile should
// import from here. Hand-maintained until apps/api ships oapi-codegen.
//
// Source of truth on the server side: apps/api/models/models.go +
// apps/api/handlers/*.go request/response shapes.

// ---------- Enums ----------

export type Verdict =
  | 'verified'
  | 'false'
  | 'misleading'
  | 'partially_true'
  | 'unverifiable'
  | 'satire'
  | 'no_claim'

export type CheckStatus = 'pending' | 'processing' | 'completed' | 'error' | 'timeout'
export type TrustTier = 'tier1' | 'tier2' | 'tier3' | 'unknown'
export type Plan = 'anon' | 'free' | 'byok' | 'plus' | 'admin'
export type Provider = 'mistral' | 'openai' | 'anthropic' | 'openrouter'

// Friendlier label aliases used by the M1 verdict copy rewrite. See
// apps/web/src/design/tokens.ts for the canonical mapping.
export type VerdictShortLabel =
  | 'WELL-SUPPORTED'
  | 'CONTRADICTED'
  | 'PARTIALLY TRUE'
  | 'MISLEADING'
  | 'NOT ENOUGH EVIDENCE'
  | 'SATIRE'
  | 'NO CLAIM'

export type ConfidenceBand = 'high' | 'medium' | 'low'

// ---------- Users / auth ----------

export interface UserInfo {
  id: number
  email: string
  plan: Plan
  emailVerifiedAt?: string
  createdAt: string
}

export interface MeResponse {
  user: UserInfo
  csrfToken: string
}

// ---------- /api/check (Phase 2) ----------

export interface DissentEntry {
  position: number
  style: string
  verdict: Verdict
  confidence: number
  errored?: boolean
}

export interface JudgeOutput {
  verdict: Verdict
  confidence: number
  summary: string
  reasoning: string
  cited_urls: string[]
  dissent_acknowledged: boolean
  steelman_for_claim?: string
  intent_interpretation?: string
}

export interface ClaimResult {
  claim: {
    id: string
    position: number
    raw_text: string
    canonical_text: string
    claim_hash: string
    difficulty: string
    high_stakes: boolean
  }
  final: JudgeOutput
  dissent: DissentEntry[]
  skeptical_fallback?: boolean
  pre_floor?: JudgeOutput
  cache_hit?: boolean
}

export interface CheckDonePayload {
  id: string
  overall_verdict: Verdict
  overall_confidence: number
  summary: string
  claims: ClaimResult[]
  byok_used: boolean
  byok_fellback: boolean
  truncated: boolean
  elapsed_ms: number
  tokens_in?: number
  tokens_out?: number
  cost_micros?: number
}

export interface CheckRow {
  id: string
  user_id?: number | null
  input_url: string
  input_caption: string
  status: CheckStatus
  plan_at_request: Plan
  fanout_n: number
  byok_used: boolean
  byok_fellback: boolean
  error?: string | null
  total_tokens_in: number
  total_tokens_out: number
  total_cost_micros: number
  truncated: boolean
  created_at: string
  started_at?: string | null
  completed_at?: string | null
}

export type SSEStage =
  | 'init'
  | 'resolve'
  | 'media'
  | 'extract'
  | 'claims'
  | 'screen'
  | 'retrieval'
  | 'retrieval_failed'
  | 'pool'
  | 'fanout'
  | 'investigator'
  | 'judge'
  | 'cache_hit'
  | 'claim_done'
  | 'byok_fallback'
  | 'done'
  | 'error'

export interface SSEEvent {
  seq: number
  stage: SSEStage
  progress: number
  message?: string
  claim_id?: string
  payload?: Record<string, unknown>
  ts: string
}

// ---------- /api/sources (Phase 3) ----------

export interface Source {
  id: number
  domain: string
  name: string
  category: string
  country?: string
  trustTier: TrustTier
}

export interface SourcesResponse {
  sources: Source[]
  count: number
}

// ---------- /api/me/journalist-application (Phase 3) ----------

export interface JournalistApplication {
  id: number
  userId: number
  fullName: string
  outlet: string
  outletUrl: string
  bylineUrls: string[]
  country?: string
  beat?: string
  bio?: string
  status: 'pending' | 'approved' | 'rejected' | 'withdrawn'
  createdAt: string
}

// ---------- /api/me/keys (Phase 2 BYOK) ----------

export interface BYOKKey {
  provider: Provider
  label: string
  lastUsedAt?: string
}

// ---------- /api/economics + /api/me/billing (Phase 5) ----------

export interface EconomicsSnapshot {
  spend_today_eur: number
  checks_today: number
  byok_percent: number
}

export interface CheckoutResponse {
  url: string
}

// ---------- Error envelope ----------

export interface ApiError {
  error: {
    code: string
    message: string
  }
}

export class ApiException extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiException'
    this.status = status
    this.code = code
  }
}
