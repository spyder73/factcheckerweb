package factcheck

import (
	"context"
	"encoding/json"
	"fmt"

	"alethea/api/models"
	"alethea/api/promptsafe"
	"alethea/api/services/ai"
)

const screenerSystem = `You are the screening pass of an evidence-based fact-checker.

Your job: given one claim, produce a quick, calibrated hint that the downstream investigators and judge will use to decide how deep to dig.

` + promptsafe.SystemBoilerplate + `

Output a SINGLE JSON object with these fields:
{
  "normalized_claim": "the claim in its clearest, most testable form, third person, no surrounding rhetoric",
  "verdict_hint": "verified" | "false" | "misleading" | "partially_true" | "unverifiable" | "satire" | "no_claim",
  "confidence_hint": 0.0..1.0,
  "difficulty": "easy" | "medium" | "hard",
  "uncertainty_questions": ["specific factual questions whose answers would resolve the claim"]
}

Be honest about uncertainty. If you'd genuinely need to look something up, say "unverifiable" with a low confidence_hint and list the lookup questions.`

type screenerResponse struct {
	NormalizedClaim       string   `json:"normalized_claim"`
	VerdictHint           string   `json:"verdict_hint"`
	ConfidenceHint        float64  `json:"confidence_hint"`
	Difficulty            string   `json:"difficulty"`
	UncertaintyQuestions  []string `json:"uncertainty_questions"`
}

// runScreener produces a ScreenerHint for one claim. Cheap-model call.
func runScreener(ctx context.Context, prov ai.Provider, claim string) (ScreenerHint, ai.Usage, []byte, error) {
	user := promptsafe.Wrap(claim, promptsafe.WrapOptions{Label: "claim under review"})
	body, usage, err := prov.CompleteJSON(ctx, screenerSystem, user, ai.CompleteOpts{
		Temperature: 0.2,
		MaxTokens:   600,
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
	}
	return hint, usage, body, nil
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
