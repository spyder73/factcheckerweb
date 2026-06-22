package factcheck

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"alethea/api/services/ai"
)

// mediaAnalysisPrompt instructs the vision model to extract factually
// relevant text + observations from one image. Kept short to limit per-image
// token spend. The output gets fed into the claim extractor.
const mediaAnalysisPrompt = `Describe this image in 2-4 sentences. Focus ONLY on facts the viewer could plausibly check:
- All visible text (verbatim if short)
- Identifiable people, places, organizations, dates
- Charts/numbers (read them exactly)
- Suspicious or implausible elements (e.g. clear photo manipulation)

Skip aesthetic description. If the image is decorative with no factual content, say "Decorative — no factual content".`

// MediaAnalysis is one image's textual summary.
type MediaAnalysis struct {
	Position int    `json:"position"`
	Source   string `json:"source"`        // short label, e.g. "image 0"
	Text     string `json:"text"`
	Errored  bool   `json:"errored,omitempty"`
}

// runMediaAnalysis sends each media item through the vision-capable provider
// in parallel. Returns per-image analyses in original order, with errors
// captured inline so one bad image doesn't poison the rest.
//
// `maxItems` caps how many images get analyzed (cost containment).
// `concurrency` caps how many analyze in parallel (rate-limit containment).
// Defaults (6, 2) are tuned for Mistral free tier (~1 req/sec). Raise both
// once on a paid plan or OpenRouter; lower to 1/1 for very strict tiers.
func runMediaAnalysis(ctx context.Context, prov ai.Provider, mediaURLs []string, maxItems, concurrency int) []MediaAnalysis {
	if !prov.SupportsVision() || len(mediaURLs) == 0 {
		return nil
	}
	if maxItems <= 0 {
		maxItems = 6
	}
	if concurrency <= 0 {
		concurrency = 2
	}
	items := mediaURLs
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	out := make([]MediaAnalysis, len(items))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, u := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, u string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					out[i] = MediaAnalysis{Position: i, Source: fmt.Sprintf("image %d", i),
						Errored: true, Text: "internal error analysing image"}
				}
			}()
			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			// AnalyzeImage doesn't take ctx in the legacy interface; we wrap
			// the call so a hung upstream still respects our overall deadline.
			// Done via a tiny goroutine + select.
			type res struct {
				text string
				err  error
			}
			ch := make(chan res, 1)
			go func() {
				txt, err := prov.AnalyzeImage(u, mediaAnalysisPrompt)
				ch <- res{txt, err}
			}()
			select {
			case <-callCtx.Done():
				out[i] = MediaAnalysis{Position: i, Source: fmt.Sprintf("image %d", i),
					Errored: true, Text: "analysis timed out"}
			case r := <-ch:
				if r.err != nil {
					out[i] = MediaAnalysis{Position: i, Source: fmt.Sprintf("image %d", i),
						Errored: true, Text: "analysis failed: " + truncateMsg(r.err.Error(), 200)}
					return
				}
				out[i] = MediaAnalysis{Position: i, Source: fmt.Sprintf("image %d", i),
					Text: strings.TrimSpace(r.text)}
			}
		}(i, u)
	}
	wg.Wait()
	return out
}

func truncateMsg(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
