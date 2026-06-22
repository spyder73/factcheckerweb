package factcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"alethea/api/auth"
	"alethea/api/models"
	"alethea/api/promptsafe"
	"alethea/api/services/ai"
	"alethea/api/services/byokresolver"
	"alethea/api/services/factcheck/cache"
	"alethea/api/services/factcheck/checkstream"
	scraperpkg "alethea/api/services"
	"alethea/api/services/factcheck/persist"
	"alethea/api/services/search"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps bundles everything the pipeline needs.
type Deps struct {
	DB                 *pgxpool.Pool
	Resolver           *byokresolver.Resolver
	Search             *search.Multi
	Cache              *cache.Cache
	Hub                *checkstream.Hub
	Scraper            *scraperpkg.ScraperService
	Reranker           Reranker // optional — nil falls back to no rerank
	OverallTimeout     time.Duration
	MaxClaims          int
	MaxQueriesPerClaim int
	HitsPerQuery       int
}

func (d *Deps) defaults() {
	if d.OverallTimeout == 0 {
		d.OverallTimeout = 90 * time.Second
	}
	if d.MaxClaims == 0 {
		d.MaxClaims = 6
	}
	if d.MaxQueriesPerClaim == 0 {
		d.MaxQueriesPerClaim = 2
	}
	if d.HitsPerQuery == 0 {
		d.HitsPerQuery = 4
	}
}

type Pipeline struct {
	deps Deps
}

func NewPipeline(d Deps) *Pipeline {
	d.defaults()
	return &Pipeline{deps: d}
}

// Run executes a full check. Designed to be called in a goroutine from the
// HTTP handler; all progress flows through the Hub channel.
func (p *Pipeline) Run(parentCtx context.Context, checkID uuid.UUID, in CheckInput, sess *auth.Session, planAtRequest string) {
	// `ctx` is the per-pipeline deadline. `cleanupCtx` survives the timeout
	// so the final DB UPDATE + SSE event always happens, even when ctx fires
	// mid-flight (without it, "completed_at" / "timeout" / "error" status
	// could remain "processing" forever on slow judge calls).
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	ctx, cancel := context.WithTimeout(parentCtx, p.deps.OverallTimeout)
	defer cancel()

	ch := p.deps.Hub.Get(checkID)
	defer ch.Close()
	emit := func(stage string, progress int, msg string, payload map[string]any) {
		ch.Emit(checkstream.Event{Stage: sanitizeStage(stage), Progress: progress, Message: msg, Payload: payload})
	}
	emit("init", 1, "starting", nil)

	_ = persist.MarkProcessing(ctx, p.deps.DB, checkID)

	keys := p.deps.Resolver.For(ctx, sess)
	defer keys.Cleanup()
	if keys.Fellback() {
		emit("byok_fallback", 2, "your BYOK key wasn't usable for this request — falling back to Alethea's pooled model", nil)
	}

	t0 := time.Now()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("pipeline panic", "check_id", checkID, "recover", fmt.Sprintf("%v", r))
			p.fail(ctx, checkID, ch, fmt.Errorf("internal error"))
		}
	}()

	// --- scrape ---
	emit("resolve", 5, "fetching content", nil)
	content, err := p.deps.Scraper.ScrapePostCtx(ctx, in.URL)
	if err != nil {
		p.fail(ctx, checkID, ch, err)
		return
	}
	if in.Caption != "" {
		content.Caption = in.Caption
	}

	// --- media analysis (optional) ---
	var mediaAnalyses []MediaAnalysis
	if len(content.MediaURLs) > 0 && keys.Judge().SupportsVision() {
		emit("media", 12, fmt.Sprintf("analyzing %d image(s)", min(len(content.MediaURLs), 6)),
			map[string]any{"count": min(len(content.MediaURLs), 6)})
		mediaAnalyses = runMediaAnalysis(ctx, keys.Judge(), content.MediaURLs, 6)
	}

	// --- claim extraction ---
	emit("extract", 20, "extracting atomic claims", nil)
	claims, err := p.extractClaims(ctx, content, mediaAnalyses, keys.Screener())
	if err != nil {
		p.fail(ctx, checkID, ch, err)
		return
	}
	if len(claims) > p.deps.MaxClaims {
		claims = claims[:p.deps.MaxClaims]
	}
	if len(claims) == 0 {
		p.completeNoClaims(ctx, checkID, ch, keys, t0, content)
		return
	}
	emit("claims", 30, fmt.Sprintf("extracted %d claim(s)", len(claims)),
		map[string]any{"count": len(claims)})

	// --- per-claim fanout ---
	claimResults := make([]ClaimResult, len(claims))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // limit concurrency across claims
	for i := range claims {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			// Per-goroutine recover — Go panics are goroutine-local; the
			// outer recover() in Run cannot catch them. Without this a nil
			// deref in processClaim would crash the entire API process.
			defer func() {
				if r := recover(); r != nil {
					slog.Error("pipeline: per-claim goroutine panic", "check_id", checkID, "claim", i, "recover", fmt.Sprintf("%v", r))
					claimResults[i] = ClaimResult{
						Claim: claims[i],
						Final: JudgeOutput{
							Verdict: models.VerdictUnverifiable, Confidence: 0,
							Summary: "An internal error prevented this claim from being evaluated.",
						},
					}
				}
			}()
			claimResults[i] = p.processClaim(ctx, checkID, claims[i], keys, planAtRequest, ch)
		}(i)
	}
	wg.Wait()

	// --- assemble ---
	overall, confidence, summary := aggregateClaims(claimResults)
	tokensIn, tokensOut, costMicros := totalCostFromDB(cleanupCtx, p.deps.DB, checkID)
	out := CheckResult{
		ID: checkID, Claims: claimResults,
		OverallVerdict: overall, OverallConfidence: confidence, Summary: summary,
		BYOKUsed:        !keys.Fellback() && hasBYOK(claimResults),
		BYOKFellback:    keys.Fellback(),
		Truncated:       ctx.Err() != nil,
		Elapsed:         time.Since(t0),
		TotalTokens:     tokensIn + tokensOut,
		TotalCostMicros: costMicros,
	}

	emit("done", 100, "", map[string]any{
		"id":                 out.ID.String(),
		"overall_verdict":    out.OverallVerdict,
		"overall_confidence": out.OverallConfidence,
		"summary":            out.Summary,
		"claims":             out.Claims,
		"byok_used":          out.BYOKUsed,
		"byok_fellback":      out.BYOKFellback,
		"truncated":          out.Truncated,
		"elapsed_ms":         out.Elapsed.Milliseconds(),
		"tokens_in":          tokensIn,
		"tokens_out":         tokensOut,
		"cost_micros":        costMicros,
	})

	status := "completed"
	if out.Truncated {
		status = "timeout"
	}
	// Use cleanupCtx so a timed-out per-check ctx doesn't prevent the row from being closed.
	_ = persist.CompleteCheck(cleanupCtx, p.deps.DB, checkID, status,
		out.BYOKUsed, out.BYOKFellback, out.Truncated, tokensIn, tokensOut, costMicros, nil)
}

// totalCostFromDB sums the per-claim agent_runs rows for this check.
// Using the DB as the source of truth means cost is correct even when an
// in-memory counter would have been racy across goroutines.
func totalCostFromDB(ctx context.Context, db *pgxpool.Pool, checkID uuid.UUID) (tokIn, tokOut, micros int) {
	row := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(ar.tokens_in),0), COALESCE(SUM(ar.tokens_out),0), COALESCE(SUM(ar.cost_micros),0)
		   FROM agent_runs ar JOIN claims c ON c.id = ar.claim_id
		  WHERE c.check_id = $1`,
		checkID)
	_ = row.Scan(&tokIn, &tokOut, &micros)
	return
}

// safeMetaField defangs newlines + caps length so a scraper-injected
// "author" or "url" can't reshape the prompt.
func safeMetaField(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 500 {
		s = s[:500] + "…"
	}
	return s
}

// min returns the smaller of two ints. Stdlib `min` requires Go 1.21+ generics
// but is available — wrap so we keep the call sites readable.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sanitizeStage strips any \r, \n, or NUL that would break SSE framing.
// Pipeline-internal callers pass constants, but the value flows verbatim
// into the `event: %s` SSE line — a defensive pass is cheap.
func sanitizeStage(s string) string {
	if s == "" {
		return "info"
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' || c == '\n' || c == 0 {
			continue
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return "info"
	}
	return string(out)
}

const extractSystem = `You extract testable factual claims from a social-media post and its image analysis.

` + promptsafe.SystemBoilerplate + `

Return a SINGLE JSON object:
{ "claims": ["claim 1 as a self-contained sentence", "claim 2", ...] }

Rules:
- Each claim must be checkable against external evidence. Opinions, jokes, and aesthetic statements are NOT claims.
- Combine related facts only if they can't be evaluated separately.
- Maximum 6 claims. If there's nothing checkable, return {"claims": []}.
- Use the third person — strip "I think", "they say", etc.`

// extractClaims runs the cheap-model condense pass. Takes the scraped caption
// plus optional textual media analyses (NEVER the raw image data — those are
// base64 strings that explode the prompt past any context window).
func (p *Pipeline) extractClaims(ctx context.Context, content *models.ContentInfo, mediaAnalyses []MediaAnalysis, prov ai.Provider) ([]Claim, error) {
	var sb strings.Builder
	sb.WriteString("POST METADATA:\n")
	fmt.Fprintf(&sb, "  platform: %s\n  author: %s\n  url: %s\n",
		safeMetaField(content.Platform), safeMetaField(content.Author), safeMetaField(content.URL))
	sb.WriteString("\nPOST CAPTION:\n")
	sb.WriteString(promptsafe.Wrap(content.Caption, promptsafe.WrapOptions{Label: "caption", MaxBytes: 8192}))
	if n := len(mediaAnalyses); n > 0 {
		fmt.Fprintf(&sb, "\n\nIMAGE ANALYSES (%d image(s)):\n", n)
		for _, m := range mediaAnalyses {
			if m.Errored {
				fmt.Fprintf(&sb, "  - %s: [ERROR] %s\n", m.Source, m.Text)
				continue
			}
			fmt.Fprintf(&sb, "  - %s: %s\n", m.Source,
				promptsafe.Wrap(m.Text, promptsafe.WrapOptions{Label: m.Source, MaxBytes: 2048}))
		}
	}
	body, _, err := prov.CompleteJSON(ctx, extractSystem, sb.String(), ai.CompleteOpts{
		Temperature: 0.1,
		MaxTokens:   800,
	})
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	var parsed struct {
		Claims []string `json:"claims"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("extract: parse: %w", err)
	}
	out := make([]Claim, 0, len(parsed.Claims))
	for i, raw := range parsed.Claims {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		canon := CanonicalizeClaim(raw)
		out = append(out, Claim{
			Position:      i,
			RawText:       raw,
			CanonicalText: canon,
			ClaimHash:     HashClaim(raw),
			Difficulty:    "medium",
			HighStakes:    IsHighStakes(raw),
		})
	}
	return out, nil
}

func (p *Pipeline) processClaim(
	ctx context.Context,
	checkID uuid.UUID,
	c Claim,
	keys byokresolver.RequestKeys,
	planAtRequest string,
	ch *checkstream.Channel,
) ClaimResult {
	emit := func(stage string, progress int, msg string, payload map[string]any) {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["claim_position"] = c.Position
		ch.Emit(checkstream.Event{Stage: stage, Progress: progress, ClaimID: c.ID.String(), Message: msg, Payload: payload})
	}

	claimRow := persist.Claim{
		CheckID:       checkID,
		Position:      c.Position,
		RawText:       c.RawText,
		CanonicalText: c.CanonicalText,
		ClaimHash:     c.ClaimHash,
		Difficulty:    c.Difficulty,
		HighStakes:    c.HighStakes,
	}
	claimID, err := persist.InsertClaim(ctx, p.deps.DB, claimRow)
	if err == nil {
		c.ID = claimID
	}

	judgeProv := keys.Judge()
	fanoutN := FanoutFor(planAtRequest)
	if hit, ok, _ := p.deps.Cache.Get(ctx, judgeProv.ModelID(), c.ClaimHash, fanoutN); ok {
		emit("cache_hit", 80, "served from cache", map[string]any{"verdict": hit.Verdict})
		final := JudgeOutput{
			Verdict:    models.Verdict(hit.Verdict),
			Confidence: hit.Confidence,
			Summary:    hit.Summary,
		}
		var dissent []DissentEntry
		_ = json.Unmarshal(hit.DissentJSON, &dissent)
		_, _ = persist.UpsertVerdict(ctx, p.deps.DB, persist.Verdict{
			ClaimID: c.ID, FinalVerdict: hit.Verdict, Confidence: hit.Confidence, Summary: hit.Summary,
			JudgeModel: hit.JudgeModel, DissentJSON: hit.DissentJSON, ContentHash: hit.ContentHash,
		})
		_, _ = p.deps.DB.Exec(ctx, `UPDATE claims SET cache_hit = TRUE WHERE id = $1`, c.ID)
		return ClaimResult{Claim: c, Final: final, Dissent: dissent, ContentHash: hit.ContentHash, CacheHit: true}
	}

	// --- screener ---
	emit("screen", 35, "screening", nil)
	hint, screenerUsage, screenerBody, err := runScreener(ctx, keys.Screener(), c.RawText)
	if err != nil {
		slog.Warn("screener failed", "claim_id", c.ID, "err", err)
	}
	if hint.NormalizedText == "" {
		hint.NormalizedText = c.RawText
	}
	hint.NormalizedText = strings.TrimSpace(hint.NormalizedText)
	verdictStr := string(hint.VerdictHint)
	conf := hint.ConfidenceHint
	_, _ = persist.InsertAgentRun(ctx, p.deps.DB, persist.AgentRun{
		ClaimID: c.ID, Role: "screener",
		Provider: keys.Screener().Name(), Model: keys.Screener().ModelID(),
		Verdict: &verdictStr, Confidence: &conf,
		TranscriptJSON: mustJSON(map[string]any{"response": string(screenerBody)}),
		TokensIn: screenerUsage.InputTokens, TokensOut: screenerUsage.OutputTokens, CostMicros: screenerUsage.CostMicros,
	})

	// --- retrieval ---
	emit("retrieval", 45, "gathering sources", nil)
	pool, err := runRetrieval(ctx, p.deps.Search, hint, p.deps.MaxQueriesPerClaim, p.deps.HitsPerQuery)
	if err != nil {
		emit("retrieval_failed", 50, "search unavailable", nil)
		return ClaimResult{
			Claim: c, Final: JudgeOutput{
				Verdict: models.VerdictUnverifiable, Confidence: 0,
				Summary: "Web search was unavailable; this claim could not be evaluated against external evidence.",
			},
		}
	}
	// Curated-source rerank: bring tier1/tier2 to the top of the pool so
	// investigators see them first. Tier lookup keyed by URL flows into
	// citation persistence so the API response can show "Vetted source" badges.
	var tierByDomain map[string]string
	if p.deps.Reranker != nil {
		pool, tierByDomain = p.deps.Reranker.Rerank(ctx, pool)
	}
	emit("pool", 55, fmt.Sprintf("source pool ready (%d sources, %d domains)", len(pool), DistinctDomainCount(pool)),
		map[string]any{"sources": len(pool), "domains": DistinctDomainCount(pool)})

	confidenceCap := 1.0
	if DistinctDomainCount(pool) < 2 {
		confidenceCap = 0.55
	}

	// --- fanout ---
	n := fanoutN
	emit("fanout", 60, fmt.Sprintf("dispatching %d investigators", n), map[string]any{"n": n})
	reports := make([]InvestigatorReport, n)
	var fwg sync.WaitGroup
	for i := 0; i < n; i++ {
		fwg.Add(1)
		go func(i int) {
			defer fwg.Done()
			// Per-investigator recover — same rationale as the per-claim recover above.
			defer func() {
				if r := recover(); r != nil {
					slog.Error("pipeline: per-investigator goroutine panic", "claim_id", c.ID, "position", i, "recover", fmt.Sprintf("%v", r))
					reports[i] = InvestigatorReport{
						Position: i, Style: styleFor(i),
						Errored: true, ErrorMessage: "internal error (panic)",
						Verdict: models.VerdictUnverifiable,
					}
				}
			}()
			style := styleFor(i)
			prov := keys.Investigator(i)
			byok := !keys.Fellback() && prov.Name() != keys.Screener().Name()
			rep, usage, body, userPrompt := runInvestigator(ctx, prov, i, style, c, pool, byok)
			reports[i] = rep

			vstr := string(rep.Verdict)
			cf := rep.Confidence
			styleCopy := style
			errMsg := rep.ErrorMessage
			runRow := persist.AgentRun{
				ClaimID: c.ID, Role: "investigator", Style: &styleCopy, Position: i,
				Provider: prov.Name(), Model: prov.ModelID(), BYOK: byok,
				Verdict: &vstr, Confidence: &cf, Reasoning: &rep.Reasoning,
				TranscriptJSON: mustJSON(map[string]any{"user_prompt": string(userPrompt), "response": string(body)}),
				TokensIn: usage.InputTokens, TokensOut: usage.OutputTokens,
				CostMicros: usage.CostMicros, DurationMs: rep.DurationMs,
				Errored: rep.Errored,
			}
			if rep.Errored {
				runRow.ErrorMessage = &errMsg
			}
			runID, perr := persist.InsertAgentRun(ctx, p.deps.DB, runRow)
			if perr == nil {
				for _, citedURL := range rep.CitedURLs {
					dom := search.ExtractDomain(citedURL)
					tier := "unknown"
					if t, ok := tierByDomain[dom]; ok {
						tier = t
					}
					_ = persist.InsertCitation(ctx, p.deps.DB, persist.Citation{
						AgentRunID: runID, URL: citedURL,
						Domain: dom, TrustTier: tier,
					})
				}
			}
		}(i)
	}
	fwg.Wait()

	// --- judge ---
	emit("judge", 80, "synthesizing", nil)
	judgeOut, judgeUsage, judgeBody, judgeUserPrompt := runJudge(ctx, judgeProv, c, hint, reports, pool)
	if judgeOut.Confidence > confidenceCap {
		judgeOut.Confidence = confidenceCap
	}
	preFloor := judgeOut
	final, demoted, _ := ApplySkepticalFloor(judgeOut)
	dissent := BuildDissent(reports)

	// content_hash inputs. We strip transient/non-deterministic fields
	// (DurationMs, Provider/Model strings that vary with deploys) so a
	// re-run with the same prompts + same evidence produces the same hash.
	var transcripts [][]byte
	for _, r := range reports {
		stable := struct {
			Position   int            `json:"position"`
			Style      string         `json:"style"`
			Verdict    models.Verdict `json:"verdict"`
			Confidence float64        `json:"confidence"`
			Reasoning  string         `json:"reasoning"`
			CitedURLs  []string       `json:"cited_urls"`
			Errored    bool           `json:"errored,omitempty"`
		}{r.Position, r.Style, r.Verdict, r.Confidence, r.Reasoning, r.CitedURLs, r.Errored}
		tb, _ := json.Marshal(stable)
		transcripts = append(transcripts, tb)
	}
	jb, _ := json.Marshal(struct {
		Verdict             models.Verdict `json:"verdict"`
		Confidence          float64        `json:"confidence"`
		Summary             string         `json:"summary"`
		Reasoning           string         `json:"reasoning"`
		CitedURLs           []string       `json:"cited_urls"`
		DissentAcknowledged bool           `json:"dissent_acknowledged,omitempty"`
		IntentInterpretation string        `json:"intent,omitempty"`
	}{final.Verdict, final.Confidence, final.Summary, final.Reasoning, final.CitedURLs, final.DissentAcknowledged, final.IntentInterpretation})
	transcripts = append(transcripts, jb)
	citedURLs := append([]string(nil), final.CitedURLs...)
	for _, r := range reports {
		citedURLs = append(citedURLs, r.CitedURLs...)
	}
	contentHash := ContentHash(c.CanonicalText, citedURLs, transcripts)

	vstr := string(final.Verdict)
	cf := final.Confidence
	_, _ = persist.InsertAgentRun(ctx, p.deps.DB, persist.AgentRun{
		ClaimID: c.ID, Role: "judge",
		Provider: judgeProv.Name(), Model: judgeProv.ModelID(), BYOK: !keys.Fellback(),
		Verdict: &vstr, Confidence: &cf, Reasoning: &final.Reasoning,
		TranscriptJSON: mustJSON(map[string]any{"user_prompt": string(judgeUserPrompt), "response": string(judgeBody)}),
		TokensIn: judgeUsage.InputTokens, TokensOut: judgeUsage.OutputTokens, CostMicros: judgeUsage.CostMicros,
	})
	dissentJSON, _ := json.Marshal(dissent)
	var preV *string
	var preC *float64
	if demoted {
		pv := string(preFloor.Verdict)
		pc := preFloor.Confidence
		preV, preC = &pv, &pc
	}
	_, _ = persist.UpsertVerdict(ctx, p.deps.DB, persist.Verdict{
		ClaimID: c.ID, FinalVerdict: vstr, Confidence: cf, Summary: final.Summary,
		JudgeModel: judgeProv.ModelID(), DissentJSON: dissentJSON,
		SkepticalFallback: demoted, PreFloorVerdict: preV, PreFloorConfidence: preC,
		ContentHash: contentHash,
	})
	_ = p.deps.Cache.Put(ctx, judgeProv.ModelID(), c.ClaimHash, fanoutN, cache.Entry{
		Verdict: vstr, Confidence: cf, Summary: final.Summary,
		JudgeModel: judgeProv.ModelID(), ContentHash: contentHash, DissentJSON: dissentJSON,
	})

	emit("claim_done", 95, fmt.Sprintf("verdict=%s confidence=%.2f", final.Verdict, final.Confidence),
		map[string]any{"verdict": final.Verdict, "confidence": final.Confidence})

	result := ClaimResult{Claim: c, Final: final, Dissent: dissent, SkepticalFallback: demoted, ContentHash: contentHash}
	if demoted {
		result.PreFloor = &preFloor
	}
	return result
}

func (p *Pipeline) fail(ctx context.Context, checkID uuid.UUID, ch *checkstream.Channel, err error) {
	msg := err.Error()
	ch.Emit(checkstream.Event{Stage: "error", Progress: 100, Message: msg})
	_ = persist.CompleteCheck(ctx, p.deps.DB, checkID, "error", false, false, false, 0, 0, 0, &msg)
}

func (p *Pipeline) completeNoClaims(ctx context.Context, checkID uuid.UUID, ch *checkstream.Channel, keys byokresolver.RequestKeys, t0 time.Time, content *models.ContentInfo) {
	out := CheckResult{
		ID: checkID, OverallVerdict: models.VerdictNoClaim, OverallConfidence: 1.0,
		Summary: "No factual claim found in the post.", Claims: nil,
		BYOKFellback: keys.Fellback(), Elapsed: time.Since(t0),
	}
	ch.Emit(checkstream.Event{Stage: "done", Progress: 100, Payload: map[string]any{
		"id": out.ID.String(), "overall_verdict": out.OverallVerdict,
		"overall_confidence": out.OverallConfidence, "summary": out.Summary,
		"claims": []any{}, "byok_fellback": out.BYOKFellback,
		"elapsed_ms": out.Elapsed.Milliseconds(),
	}})
	_ = persist.CompleteCheck(ctx, p.deps.DB, checkID, "completed", false, keys.Fellback(), false, 0, 0, 0, nil)
}

// aggregateClaims picks the worst per-claim verdict as the overall, with the
// minimum confidence across claims. Verdict severity (high → low):
// false > misleading > partially_true > unverifiable > satire > verified > no_claim.
func aggregateClaims(rs []ClaimResult) (models.Verdict, float64, string) {
	if len(rs) == 0 {
		return models.VerdictNoClaim, 1.0, "No factual claim found."
	}
	sev := map[models.Verdict]int{
		models.VerdictFalse:        6,
		models.VerdictMisleading:   5,
		models.VerdictPartialTrue:  4,
		models.VerdictUnverifiable: 3,
		models.VerdictSatire:       2,
		models.VerdictVerified:     1,
		models.VerdictNoClaim:      0,
	}
	worst := rs[0]
	minConf := rs[0].Final.Confidence
	for _, r := range rs[1:] {
		if sev[r.Final.Verdict] > sev[worst.Final.Verdict] {
			worst = r
		}
		if r.Final.Confidence < minConf {
			minConf = r.Final.Confidence
		}
	}
	summary := worst.Final.Summary
	if len(rs) > 1 {
		summary = fmt.Sprintf("Across %d claim(s), the strongest verdict was %s. %s",
			len(rs), worst.Final.Verdict, worst.Final.Summary)
	}
	return worst.Final.Verdict, minConf, summary
}

func hasBYOK(rs []ClaimResult) bool {
	// Placeholder — we'd track BYOK usage via the agent_runs rows. For Phase 2
	// returning false unless the resolver said otherwise is fine.
	return false
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}
