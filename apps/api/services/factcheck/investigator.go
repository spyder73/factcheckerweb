package factcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"alethea/api/promptsafe"
	"alethea/api/services/ai"
	"alethea/api/services/search"
)

// Investigator stylistic flavors. Phase 2 ships three; the spec proposed
// five — we keep the simplest set to reduce prompt-engineering surface.
var investigatorStyles = []string{
	"empiricist",
	"skeptic",
	"historical-context",
}

// styleFor picks a stylistic role for an investigator at position i.
// Round-robin through the available styles; deterministic across runs.
func styleFor(i int) string {
	return investigatorStyles[i%len(investigatorStyles)]
}

const investigatorSystemTmpl = `You are investigator #{POS} on a multi-agent fact-checker. You investigate one claim independently — DO NOT coordinate with other investigators.

YOUR LENS: {STYLE_INSTRUCTION}

` + promptsafe.SystemBoilerplate + `

A shared source pool was retrieved up-front. Each source has an integer ID. CITE BY ID when you reference one — do not invent URLs and do not paraphrase a source's URL into your output. Only the IDs in the pool are real.

If the pool's evidence is insufficient to reach a confident verdict, return "unverifiable" with confidence reflecting how thin the evidence is. Never bluff.

Output a SINGLE JSON object:
{
  "verdict": "verified" | "false" | "misleading" | "partially_true" | "unverifiable" | "satire" | "no_claim",
  "confidence": 0.0..1.0,
  "reasoning": "2-4 sentences justifying the verdict, referencing source IDs in [brackets]",
  "cited_source_ids": [int, ...]   // subset of the pool IDs you actually relied on
}`

var styleInstructions = map[string]string{
	"empiricist":         "Weigh primary-source evidence over commentary. Prefer numbers, dates, original documents. If the claim is empirical, the evidence is empirical.",
	"skeptic":            "Default to caution. Ask what would have to be true for the claim to be wrong, and whether the pool's evidence rules that out. Distrust single-source confirmations.",
	"historical-context": "Place the claim in its temporal and contextual frame. Is it being recycled? Stripped of caveats? Look for the older source the claim trails back to.",
}

// runInvestigator runs one investigator over the shared pool, returning a report.
// Errors from the model are captured INTO the report (Errored=true) so a
// single failure doesn't poison the whole fanout.
func runInvestigator(
	ctx context.Context,
	prov ai.Provider,
	position int,
	style string,
	claim Claim,
	pool []search.Hit,
	byok bool,
) (InvestigatorReport, ai.Usage, []byte, []byte) {
	t0 := time.Now()
	report := InvestigatorReport{
		Position: position,
		Style:    style,
		Provider: prov.Name(),
		Model:    prov.ModelID(),
		BYOK:     byok,
	}

	sys := strings.ReplaceAll(investigatorSystemTmpl, "{POS}", fmt.Sprintf("%d", position))
	instruction := styleInstructions[style]
	if instruction == "" {
		instruction = styleInstructions["empiricist"]
	}
	sys = strings.ReplaceAll(sys, "{STYLE_INSTRUCTION}", instruction)

	user := buildInvestigatorUser(claim, pool)

	body, usage, err := prov.CompleteJSON(ctx, sys, user, ai.CompleteOpts{
		Temperature: 0.4,
		MaxTokens:   1200,
	})
	report.DurationMs = int(time.Since(t0).Milliseconds())
	if err != nil {
		report.Errored = true
		report.ErrorMessage = err.Error()
		report.Verdict = "unverifiable"
		return report, usage, body, []byte(user)
	}

	var parsed struct {
		Verdict        string  `json:"verdict"`
		Confidence     float64 `json:"confidence"`
		Reasoning      string  `json:"reasoning"`
		CitedSourceIDs []int   `json:"cited_source_ids"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		report.Errored = true
		report.ErrorMessage = fmt.Sprintf("parse: %v", err)
		report.Verdict = "unverifiable"
		return report, usage, body, []byte(user)
	}

	report.Verdict = coerceVerdict(parsed.Verdict)
	report.Confidence = clamp01(parsed.Confidence)
	report.Reasoning = parsed.Reasoning
	// Resolve cited IDs back to URLs against the pool. Drop any out-of-range
	// IDs (hallucination guard — pool can't be poisoned by made-up sources).
	for _, id := range parsed.CitedSourceIDs {
		if id >= 0 && id < len(pool) {
			report.CitedURLs = append(report.CitedURLs, pool[id].URL)
		}
	}
	return report, usage, body, []byte(user)
}

// buildInvestigatorUser renders the user prompt: claim + numbered pool.
func buildInvestigatorUser(claim Claim, pool []search.Hit) string {
	var sb strings.Builder
	sb.WriteString("CLAIM TO INVESTIGATE:\n")
	sb.WriteString(promptsafe.Wrap(claim.RawText, promptsafe.WrapOptions{Label: "claim"}))
	sb.WriteString("\n\nSHARED SOURCE POOL (cite by id):\n")
	for i, h := range pool {
		// Wrap EVERY third-party-derived string: title, snippet, even the URL
		// (which can contain attacker-chosen text in path/query). Domains are
		// already canonicalized to host names — safer but still wrap to be uniform.
		title := promptsafe.Wrap(h.Title, promptsafe.WrapOptions{Label: fmt.Sprintf("source %d title", i), MaxBytes: 512})
		snippet := promptsafe.Wrap(h.Snippet, promptsafe.WrapOptions{Label: fmt.Sprintf("source %d snippet", i), MaxBytes: 2048})
		urlWrap := promptsafe.Wrap(h.URL, promptsafe.WrapOptions{Label: fmt.Sprintf("source %d url", i), MaxBytes: 1024})
		fmt.Fprintf(&sb, "\n[id=%d]\n  title: %s\n  url: %s\n  domain: %s\n  snippet: %s\n",
			i, title, urlWrap, h.SourceDomain, snippet)
	}
	return sb.String()
}
