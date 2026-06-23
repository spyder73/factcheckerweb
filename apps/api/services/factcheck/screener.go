package factcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"alethea/api/models"
	"alethea/api/promptsafe"
	"alethea/api/services/ai"
)

// screenerSystemTemplate gets %s replaced with the current ISO date so the
// model can resolve relative time expressions ("today", "this week", "the
// current year") into absolute references that retrieval can act on.
// M13 from the Phase A independent review.
const screenerSystemTemplate = `You are the screening pass of an evidence-based fact-checker.

The current date is %s (UTC). Resolve relative time expressions ("today",
"yesterday", "this week", "this year") to absolute dates in normalized_claim
and in uncertainty_questions, so retrieval can search effectively.

Your job: given one claim, produce a quick, calibrated hint that the downstream investigators and judge will use to decide how deep to dig.

` + promptsafe.SystemBoilerplate + `

Output a SINGLE JSON object with these fields:
{
  "normalized_claim": "the claim in its clearest, most testable form, third person, no surrounding rhetoric, absolute dates",
  "verdict_hint": "verified" | "false" | "misleading" | "partially_true" | "unverifiable" | "satire" | "no_claim",
  "confidence_hint": 0.0..1.0,
  "difficulty": "easy" | "medium" | "hard",
  "uncertainty_questions": ["specific factual questions whose answers would resolve the claim"],
  "out_of_scope": "" | "opinion" | "prediction" | "personal_experience" | "private_individual" | "religion_or_ideology" | "humor"
}

Set "out_of_scope" to a non-empty category when the input is not a verifiable factual claim about the external world:
- "opinion" — value judgements ("X is the best", "Y is overrated"), aesthetic preferences
- "prediction" — claims about the future ("X will happen", "Y is going to lose")
- "personal_experience" — claims about the speaker's own private experience ("I felt", "I saw")
- "private_individual" — claims about a non-public person's private life
- "religion_or_ideology" — metaphysical or doctrinal claims that are not testable against evidence
- "humor" — sarcasm, jokes, obvious irony where treating the line as a factual claim would miss the point
Leave "" for ordinary verifiable claims about the world.

Be honest about uncertainty. If you'd genuinely need to look something up, say "unverifiable" with a low confidence_hint and list the lookup questions.`

func screenerSystem() string {
	return fmt.Sprintf(screenerSystemTemplate, time.Now().UTC().Format("2006-01-02"))
}

type screenerResponse struct {
	NormalizedClaim       string   `json:"normalized_claim"`
	VerdictHint           string   `json:"verdict_hint"`
	ConfidenceHint        float64  `json:"confidence_hint"`
	Difficulty            string   `json:"difficulty"`
	UncertaintyQuestions  []string `json:"uncertainty_questions"`
	OutOfScope            string   `json:"out_of_scope"`
}

// runScreener produces a ScreenerHint for one claim. Cheap-model call.
func runScreener(ctx context.Context, prov ai.Provider, claim string) (ScreenerHint, ai.Usage, []byte, error) {
	user := promptsafe.Wrap(claim, promptsafe.WrapOptions{Label: "claim under review"})
	body, usage, err := prov.CompleteJSON(ctx, screenerSystem(), user, ai.CompleteOpts{
		Temperature: 0.2,
		MaxTokens:   700,
	})
	if err != nil {
		return ScreenerHint{}, usage, nil, fmt.Errorf("screener: %w", err)
	}
	var r screenerResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return ScreenerHint{}, usage, body, fmt.Errorf("screener: parse: %w (body=%s)", err, truncateForLog(string(body)))
	}
	hint := ScreenerHint{
		VerdictHint:    coerceVerdict(r.VerdictHint),
		ConfidenceHint: clamp01(r.ConfidenceHint),
		NormalizedText: pick(r.NormalizedClaim, claim),
		Difficulty:     coerceDifficulty(r.Difficulty),
		UncertaintyQs:  r.UncertaintyQuestions,
		OutOfScope:     coerceOutOfScope(r.OutOfScope),
	}
	return hint, usage, body, nil
}

// coerceOutOfScope keeps the screener honest — only accept values from the
// known refuse-list categories; anything else collapses to empty.
func coerceOutOfScope(s string) string {
	switch s {
	case "opinion", "prediction", "personal_experience", "private_individual", "religion_or_ideology", "humor":
		return s
	}
	return ""
}

// OutOfScopeMessage is the user-visible explanation per category. Kept here
// (Go side) because the pipeline emits the verdict; the frontend just
// renders the string. Action-line phrasing per Phase A M1.
var OutOfScopeMessage = map[string]string{
	"opinion":              "This is a value judgment, not a factual claim. We don't fact-check opinions.",
	"prediction":           "This is a claim about the future. We can only fact-check claims about what is or has been.",
	"personal_experience":  "This is about the speaker's own private experience. We can't verify subjective experiences.",
	"private_individual":   "This is about a private individual's personal life. We don't fact-check claims about non-public people.",
	"religion_or_ideology": "This is a metaphysical or doctrinal claim. It's not testable against evidence.",
	"humor":                "This reads as humor or sarcasm. Treating it as a literal factual claim would miss the point.",
}

func coerceVerdict(s string) models.Verdict {
	v := models.Verdict(s)
	switch v {
	case models.VerdictVerified, models.VerdictFalse, models.VerdictMisleading,
		models.VerdictPartialTrue, models.VerdictUnverifiable, models.VerdictSatire, models.VerdictNoClaim:
		return v
	}
	return models.VerdictUnverifiable
}

func coerceDifficulty(s string) string {
	switch s {
	case "easy", "medium", "hard":
		return s
	}
	return "medium"
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func pick(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func truncateForLog(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}
