// Package persist owns the database access for the Phase 2 pipeline.
// Thin wrappers around pgx — every method takes context + pool, returns
// either a typed struct or a wrapped error. No business logic here.
package persist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- checks ---------------------------------------------------------------

type Check struct {
	ID             uuid.UUID
	UserID         *int64
	InputURL       string
	InputCaption   string
	Status         string
	PlanAtRequest  string
	FanoutN        int
	BYOKUsed       bool
	BYOKFellback   bool
	Error          *string
	TotalTokensIn  int
	TotalTokensOut int
	TotalCostMicros int
	Truncated      bool
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

// NewCheck creates a check row in 'pending' status and returns its ID.
// AnonToken is the hashed SSE-subscription token for logged-out callers
// (nil for authenticated callers — they identify via the session cookie).
func NewCheck(ctx context.Context, pool *pgxpool.Pool, userID *int64, anonTokenHash []byte, url, caption, plan string, fanoutN int) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO checks (user_id, anon_token, input_url, input_caption, plan_at_request, fanout_n)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		userID, anonTokenHash, url, caption, plan, fanoutN,
	).Scan(&id)
	return id, err
}

// MarkProcessing flips status to processing and stamps started_at.
func MarkProcessing(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE checks SET status = 'processing', started_at = NOW() WHERE id = $1 AND status = 'pending'`, id)
	return err
}

// CompleteCheck stamps completed_at and updates totals + status.
func CompleteCheck(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, byokUsed, byokFellback, truncated bool, tokensIn, tokensOut, costMicros int, errMsg *string) error {
	_, err := pool.Exec(ctx,
		`UPDATE checks
		    SET status = $2,
		        byok_used = $3,
		        byok_fellback = $4,
		        truncated = $5,
		        total_tokens_in = $6,
		        total_tokens_out = $7,
		        total_cost_micros = $8,
		        error = $9,
		        completed_at = NOW()
		  WHERE id = $1`,
		id, status, byokUsed, byokFellback, truncated, tokensIn, tokensOut, costMicros, errMsg,
	)
	return err
}

func GetCheck(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Check, error) {
	c := &Check{}
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, input_url, input_caption, status, plan_at_request, fanout_n,
		        byok_used, byok_fellback, error, total_tokens_in, total_tokens_out,
		        total_cost_micros, truncated, created_at, started_at, completed_at
		   FROM checks WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.UserID, &c.InputURL, &c.InputCaption, &c.Status, &c.PlanAtRequest, &c.FanoutN,
		&c.BYOKUsed, &c.BYOKFellback, &c.Error, &c.TotalTokensIn, &c.TotalTokensOut,
		&c.TotalCostMicros, &c.Truncated, &c.CreatedAt, &c.StartedAt, &c.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// --- claims ---------------------------------------------------------------

type Claim struct {
	ID            uuid.UUID
	CheckID       uuid.UUID
	Position      int
	RawText       string
	CanonicalText string
	ClaimHash     []byte
	Difficulty    string
	HighStakes    bool
	CacheHit      bool
	CreatedAt     time.Time
}

func InsertClaim(ctx context.Context, pool *pgxpool.Pool, c Claim) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO claims (check_id, position, raw_text, canonical_text, claim_hash, difficulty, high_stakes, cache_hit)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id`,
		c.CheckID, c.Position, c.RawText, c.CanonicalText, c.ClaimHash, c.Difficulty, c.HighStakes, c.CacheHit,
	).Scan(&id)
	return id, err
}

// --- agent_runs -----------------------------------------------------------

type AgentRun struct {
	ID             uuid.UUID
	ClaimID        uuid.UUID
	Role           string  // 'screener','retrieval','investigator','judge','intent'
	Style          *string
	Position       int
	Provider       string
	Model          string
	BYOK           bool
	Verdict        *string
	Confidence     *float64
	Reasoning      *string
	TranscriptJSON []byte
	TokensIn       int
	TokensOut      int
	CostMicros     int
	DurationMs     int
	Errored        bool
	ErrorMessage   *string
}

func InsertAgentRun(ctx context.Context, pool *pgxpool.Pool, r AgentRun) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO agent_runs
		   (claim_id, role, style, position, provider, model, byok,
		    verdict, confidence, reasoning, transcript_json,
		    tokens_in, tokens_out, cost_micros, duration_ms, errored, error_message)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14,$15,$16,$17)
		 RETURNING id`,
		r.ClaimID, r.Role, r.Style, r.Position, r.Provider, r.Model, r.BYOK,
		r.Verdict, r.Confidence, r.Reasoning, nullableJSON(r.TranscriptJSON),
		r.TokensIn, r.TokensOut, r.CostMicros, r.DurationMs, r.Errored, r.ErrorMessage,
	).Scan(&id)
	return id, err
}

// --- verdicts -------------------------------------------------------------

type Verdict struct {
	ID                  uuid.UUID
	ClaimID             uuid.UUID
	FinalVerdict        string
	Confidence          float64
	Summary             string
	JudgeModel          string
	DissentJSON         []byte
	SkepticalFallback   bool
	PreFloorVerdict     *string
	PreFloorConfidence  *float64
	ContentHash         []byte
}

func UpsertVerdict(ctx context.Context, pool *pgxpool.Pool, v Verdict) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO verdicts
		   (claim_id, final_verdict, confidence, summary, judge_model, dissent_json,
		    skeptical_fallback, pre_floor_verdict, pre_floor_confidence, content_hash)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)
		 ON CONFLICT (claim_id) DO UPDATE
		   SET final_verdict = EXCLUDED.final_verdict,
		       confidence = EXCLUDED.confidence,
		       summary = EXCLUDED.summary,
		       judge_model = EXCLUDED.judge_model,
		       dissent_json = EXCLUDED.dissent_json,
		       skeptical_fallback = EXCLUDED.skeptical_fallback,
		       pre_floor_verdict = EXCLUDED.pre_floor_verdict,
		       pre_floor_confidence = EXCLUDED.pre_floor_confidence,
		       content_hash = EXCLUDED.content_hash
		 RETURNING id`,
		v.ClaimID, v.FinalVerdict, v.Confidence, v.Summary, v.JudgeModel, v.DissentJSON,
		v.SkepticalFallback, v.PreFloorVerdict, v.PreFloorConfidence, v.ContentHash,
	).Scan(&id)
	return id, err
}

// --- citations ------------------------------------------------------------

type Citation struct {
	AgentRunID uuid.UUID
	SourceID   *int64
	URL        string
	Domain     string
	Title      *string
	Quote      *string
	TrustTier  string
	Stance     *string
}

func InsertCitation(ctx context.Context, pool *pgxpool.Pool, c Citation) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO citations (agent_run_id, source_id, url, domain, title, quote, trust_tier, stance)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.AgentRunID, c.SourceID, c.URL, c.Domain, c.Title, c.Quote, c.TrustTier, c.Stance,
	)
	return err
}

// --- byok_keys ------------------------------------------------------------

type BYOKKey struct {
	ID         int64
	UserID     int64
	Provider   string
	Label      string
	Ciphertext []byte
	Nonce      []byte
	LastUsedAt *time.Time
}

func UpsertBYOK(ctx context.Context, pool *pgxpool.Pool, userID int64, provider, label string, ciphertext, nonce []byte) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO byok_keys (user_id, provider, label, ciphertext, nonce)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id, provider) DO UPDATE
		   SET label = EXCLUDED.label,
		       ciphertext = EXCLUDED.ciphertext,
		       nonce = EXCLUDED.nonce,
		       last_used_at = NULL`,
		userID, provider, label, ciphertext, nonce,
	)
	return err
}

func GetBYOK(ctx context.Context, pool *pgxpool.Pool, userID int64, provider string) (*BYOKKey, error) {
	k := &BYOKKey{}
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, provider, label, ciphertext, nonce, last_used_at
		   FROM byok_keys WHERE user_id = $1 AND provider = $2`,
		userID, provider,
	).Scan(&k.ID, &k.UserID, &k.Provider, &k.Label, &k.Ciphertext, &k.Nonce, &k.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return k, err
}

// ListBYOK returns metadata only — the ciphertext + nonce are NOT loaded
// because the List handler doesn't decrypt and we don't want secrets in
// memory we don't have to have. Use GetBYOK when you actually need to decrypt.
func ListBYOK(ctx context.Context, pool *pgxpool.Pool, userID int64) ([]BYOKKey, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, provider, label, last_used_at
		   FROM byok_keys WHERE user_id = $1 ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BYOKKey
	for rows.Next() {
		k := BYOKKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Provider, &k.Label, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// DeleteBYOK returns ErrNotFound when no row matched, so the handler can
// correctly distinguish "your key was deleted" from "you tried to delete
// a key that wasn't there" (still 204 — the end state is identical — but
// audit-log semantics differ).
func DeleteBYOK(ctx context.Context, pool *pgxpool.Pool, userID int64, provider string) error {
	tag, err := pool.Exec(ctx, `DELETE FROM byok_keys WHERE user_id = $1 AND provider = $2`, userID, provider)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func TouchBYOK(ctx context.Context, pool *pgxpool.Pool, id int64) {
	_, _ = pool.Exec(ctx, `UPDATE byok_keys SET last_used_at = NOW() WHERE id = $1`, id)
}

// --- helpers --------------------------------------------------------------

var ErrNotFound = errors.New("persist: not found")

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// CompactErr makes errors marginally less noisy by tagging them with the calling op.
func CompactErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("persist.%s: %w", op, err)
}
