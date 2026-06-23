// Package factcheck owns the Phase 2 multi-agent fact-check pipeline.
package factcheck

import (
	"time"

	"alethea/api/models"
	"alethea/api/services/search"

	"github.com/google/uuid"
)

// CheckInput is what comes in from the HTTP handler.
type CheckInput struct {
	URL     string
	Caption string
	// FanoutN overrides the per-tier default. Zero = use the tier default.
	FanoutN int
}

// Claim is a single atomic claim extracted from the scraped content.
type Claim struct {
	ID            uuid.UUID
	Position      int
	RawText       string
	CanonicalText string
	ClaimHash     []byte
	Difficulty    string // 'easy','medium','hard'
	HighStakes    bool
}

// ScreenerHint is the cheap-model preview that informs retrieval + fanout sizing.
type ScreenerHint struct {
	VerdictHint    models.Verdict
	ConfidenceHint float64
	NormalizedText string
	Difficulty     string
	UncertaintyQs  []string // optional follow-up questions investigators can use
	// OutOfScope, when non-empty, marks the claim as something we should
	// refuse to fact-check rather than spend pipeline budget on. See
	// coerceOutOfScope in screener.go for the allowed category strings.
	// Phase A M12 (refuse list).
	OutOfScope string
}

// InvestigatorReport is what one investigator returns.
type InvestigatorReport struct {
	Position     int               `json:"position"`
	Style        string            `json:"style"`
	Verdict      models.Verdict    `json:"verdict"`
	Confidence   float64           `json:"confidence"`
	Reasoning    string            `json:"reasoning"`
	CitedURLs    []string          `json:"cited_urls"`
	Errored      bool              `json:"errored,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	DurationMs   int               `json:"duration_ms"`
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	BYOK         bool              `json:"byok"`
}

// JudgeOutput is the final per-claim synthesis.
type JudgeOutput struct {
	Verdict             models.Verdict `json:"verdict"`
	Confidence          float64        `json:"confidence"`
	Summary             string         `json:"summary"`
	Reasoning           string         `json:"reasoning"`
	CitedURLs           []string       `json:"cited_urls"`
	DissentAcknowledged bool           `json:"dissent_acknowledged"`
	SteelmanForClaim    string         `json:"steelman_for_claim,omitempty"`
	IntentInterpretation string        `json:"intent_interpretation,omitempty"`
}

// ClaimResult is the per-claim payload assembled in the orchestrator. The
// dissent vector is the per-investigator verdict so the frontend can render
// it as a bar chart and the API caller can pin non-repudiation.
type ClaimResult struct {
	Claim            Claim          `json:"claim"`
	Final            JudgeOutput    `json:"final"`
	Dissent          []DissentEntry `json:"dissent"`
	SkepticalFallback bool          `json:"skeptical_fallback"`
	PreFloor         *JudgeOutput   `json:"pre_floor,omitempty"`
	ContentHash      []byte         `json:"-"`
	CacheHit         bool           `json:"cache_hit"`
}

// DissentEntry is one investigator's vote, shown to the client verbatim.
type DissentEntry struct {
	Position   int            `json:"position"`
	Style      string         `json:"style"`
	Verdict    models.Verdict `json:"verdict"`
	Confidence float64        `json:"confidence"`
	Errored    bool           `json:"errored,omitempty"`
}

// CheckResult is what the orchestrator hands back to the handler.
type CheckResult struct {
	ID            uuid.UUID
	OverallVerdict models.Verdict
	OverallConfidence float64
	Summary       string
	Claims        []ClaimResult
	BYOKUsed      bool
	BYOKFellback  bool
	Truncated     bool
	TotalTokens   int
	TotalCostMicros int
	Elapsed       time.Duration
}

// SourcePool is the shared retrieval output passed to all investigators
// for one claim. Hits are de-duped by URL and assigned stable IDs (0..N-1)
// so the model can cite by ID rather than echoing back URLs.
type SourcePool struct {
	ClaimID uuid.UUID
	Hits    []search.Hit
}

// FanoutFor returns the per-tier default N.
func FanoutFor(plan string) int {
	switch plan {
	case "plus":
		return 5
	case "byok":
		return 3
	case "admin":
		return 5
	default:
		return 1
	}
}
