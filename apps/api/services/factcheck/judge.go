package factcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"alethea/api/models"
	"alethea/api/promptsafe"
	"alethea/api/services/ai"
	"alethea/api/services/search"
)

const judgeSystem = `You are the JUDGE of a multi-agent fact-checker. You synthesize N independent investigator reports plus their shared source pool into ONE final verdict for the user.

` + promptsafe.SystemBoilerplate + `

GROUND RULES:
- Confidence below 0.65 ⇒ verdict MUST be "unverifiable", not a guess.
- High dissent (investigators disagreeing on the same evidence) is a signal of genuine ambiguity. Reflect it in your confidence, not by averaging votes.
- When dissent is high, write a brief "steelman_for_claim" that states the strongest case FOR the claim. Don't editorialize it.
- Intent interpretation: write one neutral sentence about what the post seems to be trying to do (sell, persuade, mock, inform). Label uncertainty. This is interpretation, not fact.

Output a SINGLE JSON object:
{
  "verdict": "verified" | "false" | "misleading" | "partially_true" | "unverifiable" | "satire" | "no_claim",
  "confidence": 0.0..1.0,
  "summary": "one-paragraph plain-language verdict for an end user",
  "reasoning": "deeper explanation: which investigators converged, which dissented, what evidence weighed most",
  "cited_source_ids": [int, ...],
  "dissent_acknowledged": true|false,
  "steelman_for_claim": "<= 2 sentences if dissent_acknowledged",
  "intent_interpretation": "one neutral sentence"
}`

// runJudge synthesizes the final verdict and intent.
func runJudge(
	ctx context.Context,
	prov ai.Provider,
	claim Claim,
	hint ScreenerHint,
	reports []InvestigatorReport,
	pool []search.Hit,
) (JudgeOutput, ai.Usage, []byte, []byte) {
	user := buildJudgeUser(claim, hint, reports, pool)
	body, usage, err := prov.CompleteJSON(ctx, judgeSystem, user, ai.CompleteOpts{
		Temperature: 0.2,
		MaxTokens:   1600,
	})
	if err != nil {
		// Hard failure on judge → return a defensive unverifiable.
		return JudgeOutput{
			Verdict:    models.VerdictUnverifiable,
			Confidence: 0.0,
			Summary:    "The judge model failed to complete the synthesis; treating as unverifiable.",
			Reasoning:  fmt.Sprintf("judge error: %v", err),
		}, usage, body, []byte(user)
	}

	var parsed struct {
		Verdict             string  `json:"verdict"`
		Confidence          float64 `json:"confidence"`
		Summary             string  `json:"summary"`
		Reasoning           string  `json:"reasoning"`
		CitedSourceIDs      []int   `json:"cited_source_ids"`
		DissentAcknowledged bool    `json:"dissent_acknowledged"`
		Steelman            string  `json:"steelman_for_claim"`
		Intent              string  `json:"intent_interpretation"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return JudgeOutput{
			Verdict:    models.VerdictUnverifiable,
			Confidence: 0.0,
			Summary:    "The judge model returned malformed JSON; treating as unverifiable.",
			Reasoning:  fmt.Sprintf("judge parse error: %v", err),
		}, usage, body, []byte(user)
	}

	out := JudgeOutput{
		Verdict:              coerceVerdict(parsed.Verdict),
		Confidence:           clamp01(parsed.Confidence),
		Summary:              parsed.Summary,
		Reasoning:            parsed.Reasoning,
		DissentAcknowledged:  parsed.DissentAcknowledged,
		SteelmanForClaim:     parsed.Steelman,
		IntentInterpretation: parsed.Intent,
	}
	for _, id := range parsed.CitedSourceIDs {
		if id >= 0 && id < len(pool) {
			out.CitedURLs = append(out.CitedURLs, pool[id].URL)
		}
	}
	return out, usage, body, []byte(user)
}

// ApplySkepticalFloor enforces the "confidence below threshold ⇒ unverifiable"
// rule. Returns the possibly-rewritten verdict + a snapshot of the pre-floor
// values so the audit log can record what the judge originally said.
func ApplySkepticalFloor(in JudgeOutput) (out JudgeOutput, demoted bool, preFloor JudgeOutput) {
	preFloor = in
	thresholds := map[models.Verdict]float64{
		models.VerdictVerified:    0.75,
		models.VerdictFalse:       0.75,
		models.VerdictMisleading:  0.65,
		models.VerdictPartialTrue: 0.65,
	}
	floor, has := thresholds[in.Verdict]
	if !has || in.Confidence >= floor {
		return in, false, preFloor
	}
	in.Verdict = models.VerdictUnverifiable
	in.Summary = "Evidence was insufficient to make a confident determination. " + in.Summary
	return in, true, preFloor
}

func buildJudgeUser(claim Claim, hint ScreenerHint, reports []InvestigatorReport, pool []search.Hit) string {
	var sb strings.Builder
	sb.WriteString("CLAIM:\n")
	sb.WriteString(promptsafe.Wrap(claim.RawText, promptsafe.WrapOptions{Label: "claim"}))
	sb.WriteString(fmt.Sprintf("\n\nSCREENER HINT: verdict_hint=%s, confidence_hint=%.2f, difficulty=%s\n",
		hint.VerdictHint, hint.ConfidenceHint, hint.Difficulty))
	sb.WriteString("\nINVESTIGATOR REPORTS:\n")
	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("\n--- investigator %d (%s, model=%s) ---\n", r.Position, r.Style, r.Model))
		if r.Errored {
			// Even the error message comes from the AI provider — wrap it.
			errSafe := promptsafe.Wrap(r.ErrorMessage, promptsafe.WrapOptions{Label: "investigator error", MaxBytes: 1024})
			sb.WriteString(fmt.Sprintf("  [ERRORED] %s — treat this report as data missing, not as a vote.\n", errSafe))
			continue
		}
		sb.WriteString(fmt.Sprintf("  verdict=%s confidence=%.2f\n", r.Verdict, r.Confidence))
		// Reasoning is model-generated and untrusted by definition (the model
		// could echo a prompt-injection payload from the scraped content).
		reasoningSafe := promptsafe.Wrap(r.Reasoning, promptsafe.WrapOptions{Label: "investigator reasoning", MaxBytes: 4096})
		sb.WriteString(fmt.Sprintf("  reasoning: %s\n", reasoningSafe))
		sb.WriteString(fmt.Sprintf("  cited URLs (%d): see pool ids in reasoning\n", len(r.CitedURLs)))
	}
	sb.WriteString("\nSOURCE POOL (cite by id):\n")
	for i, h := range pool {
		titleSafe := promptsafe.Wrap(h.Title, promptsafe.WrapOptions{Label: fmt.Sprintf("source %d title", i), MaxBytes: 512})
		urlSafe := promptsafe.Wrap(h.URL, promptsafe.WrapOptions{Label: fmt.Sprintf("source %d url", i), MaxBytes: 1024})
		fmt.Fprintf(&sb, "[id=%d] title=%s url=%s\n", i, titleSafe, urlSafe)
	}
	return sb.String()
}

// BuildDissent returns the per-investigator vote vector the API ships back
// to the frontend (for bar-chart rendering and non-repudiation).
func BuildDissent(reports []InvestigatorReport) []DissentEntry {
	out := make([]DissentEntry, len(reports))
	for i, r := range reports {
		out[i] = DissentEntry{
			Position:   r.Position,
			Style:      r.Style,
			Verdict:    r.Verdict,
			Confidence: r.Confidence,
			Errored:    r.Errored,
		}
	}
	return out
}
