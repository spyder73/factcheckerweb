package services

import (
	"context"
	"sync"
	"time"

	"fact-checker/models"

	"github.com/google/uuid"
)

// FactCheckService orchestrates the fact-checking process
type FactCheckService struct {
	ai        *AIService
	scraper   *ScraperService
	checks    map[string]*models.CheckResponse
	listeners map[string][]chan models.ProgressUpdate
	mu        sync.RWMutex
}

// NewFactCheckService creates a new fact-check service
func NewFactCheckService(ai *AIService, scraper *ScraperService) *FactCheckService {
	return &FactCheckService{
		ai:        ai,
		scraper:   scraper,
		checks:    make(map[string]*models.CheckResponse),
		listeners: make(map[string][]chan models.ProgressUpdate),
	}
}

// StartCheck initiates a new fact-check
func (f *FactCheckService) StartCheck(req models.CheckRequest) (*models.CheckResponse, error) {
	check := models.NewCheckResponse()
	check.Status = models.StatusProcessing
	check.CurrentStep = "Initializing..."

	f.mu.Lock()
	f.checks[check.ID] = check
	f.mu.Unlock()

	// Start processing in background
	go f.processCheck(check.ID, req)

	return check, nil
}

// GetCheck returns the current state of a check
func (f *FactCheckService) GetCheck(id string) (*models.CheckResponse, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	check, ok := f.checks[id]
	return check, ok
}

// Subscribe returns a channel for progress updates
func (f *FactCheckService) Subscribe(id string) chan models.ProgressUpdate {
	ch := make(chan models.ProgressUpdate, 10)

	f.mu.Lock()
	f.listeners[id] = append(f.listeners[id], ch)
	f.mu.Unlock()

	return ch
}

// Unsubscribe removes a listener
func (f *FactCheckService) Unsubscribe(id string, ch chan models.ProgressUpdate) {
	f.mu.Lock()
	defer f.mu.Unlock()

	listeners := f.listeners[id]
	for i, l := range listeners {
		if l == ch {
			f.listeners[id] = append(listeners[:i], listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

func (f *FactCheckService) updateProgress(id string, step string, progress int, message string) {
	f.mu.Lock()
	if check, ok := f.checks[id]; ok {
		check.CurrentStep = step
		check.Progress = progress
	}
	listeners := f.listeners[id]
	f.mu.Unlock()

	update := models.ProgressUpdate{
		Step:     step,
		Progress: progress,
		Message:  message,
	}

	for _, ch := range listeners {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}

func (f *FactCheckService) processCheck(id string, req models.CheckRequest) {
	startTime := time.Now()
	ctx := context.Background()
	_ = ctx // For future cancellation support

	// Step 1: Scrape content
	f.updateProgress(id, "scraping", 10, "Fetching content from URL...")

	content, err := f.scraper.ScrapePost(req.URL)
	if err != nil {
		f.setError(id, "Failed to fetch content: "+err.Error())
		return
	}

	// Add manual caption if provided
	if req.Caption != "" && content.Caption == "" {
		content.Caption = req.Caption
	}

	// Step 2: Analyze media
	f.updateProgress(id, "analyzing_media", 25, "Analyzing images and media...")

	var mediaAnalyses []ImageAnalysis
	var mediaInfos []models.MediaInfo

	for i, mediaURL := range content.MediaURLs {
		f.updateProgress(id, "analyzing_media", 25+((i+1)*10),
			"Analyzing media "+(string(rune('1'+i)))+"...")

		analysis, err := f.ai.AnalyzeImage(mediaURL, content.Caption)
		if err != nil {
			// Continue with other media on error
			continue
		}

		mediaAnalyses = append(mediaAnalyses, analysis)
		mediaInfos = append(mediaInfos, models.MediaInfo{
			ID:          uuid.New().String(),
			Type:        "image",
			URL:         mediaURL,
			Description: analysis.Description,
			Elements:    analysis.Elements,
			TextFound:   analysis.TextFound,
		})
	}

	// If no media, analyze caption directly
	if len(mediaAnalyses) == 0 && content.Caption != "" {
		mediaAnalyses = append(mediaAnalyses, ImageAnalysis{
			Description: content.Caption,
			Claims:      []string{content.Caption},
		})
	}

	// Step 3: Condense information
	f.updateProgress(id, "condensing", 50, "Extracting key claims...")

	condensed, err := f.ai.CondenseInformation(mediaAnalyses, content.Caption)
	if err != nil {
		f.setError(id, "Failed to analyze content: "+err.Error())
		return
	}

	// Step 4: Evaluate truthfulness
	f.updateProgress(id, "evaluating", 70, "Verifying claims against sources...")

	evaluation, err := f.ai.EvaluateTruthfulness(condensed)
	if err != nil {
		f.setError(id, "Failed to evaluate claims: "+err.Error())
		return
	}

	// Step 5: Build result
	f.updateProgress(id, "finalizing", 90, "Compiling results...")

	result := f.buildResult(content, mediaInfos, condensed, evaluation)

	// Complete
	f.mu.Lock()
	if check, ok := f.checks[id]; ok {
		check.Status = models.StatusCompleted
		check.Progress = 100
		check.CurrentStep = "Complete"
		check.Result = result
		check.ProcessingTime = time.Since(startTime).Seconds()
	}
	f.mu.Unlock()

	f.updateProgress(id, "complete", 100, "Fact-check complete!")
}

func (f *FactCheckService) setError(id string, errMsg string) {
	f.mu.Lock()
	if check, ok := f.checks[id]; ok {
		check.Status = models.StatusError
		check.Error = errMsg
	}
	f.mu.Unlock()

	f.updateProgress(id, "error", 0, errMsg)
}

func (f *FactCheckService) buildResult(
	content *models.ContentInfo,
	mediaInfos []models.MediaInfo,
	condensed CondensedInfo,
	evaluation TruthEvaluation,
) *models.FactCheckResult {

	// Convert verdict
	verdict := models.Verdict(evaluation.Verdict)
	if verdict == "" {
		verdict = models.VerdictUnverifiable
	}

	// Build claims
	var claims []models.Claim
	for _, c := range evaluation.Claims {
		claims = append(claims, models.Claim{
			ID:              uuid.New().String(),
			Statement:       c.Statement,
			Verdict:         models.Verdict(c.Verdict),
			Explanation:     c.Explanation,
			ProArguments:    c.ProArguments,
			ContraArguments: c.ContraArguments,
		})
	}

	// If no claims, create one from the main message
	if len(claims) == 0 && condensed.OverallMessage != "" {
		claims = append(claims, models.Claim{
			ID:          uuid.New().String(),
			Statement:   condensed.OverallMessage,
			Verdict:     verdict,
			Explanation: evaluation.Summary,
		})
	}

	// Build sources
	var sources []models.Source
	for _, s := range evaluation.Sources {
		stance := models.Stance(s.Stance)
		if stance == "" {
			stance = models.StanceNeutral
		}

		sources = append(sources, models.Source{
			ID:          uuid.New().String(),
			Title:       s.Name,
			Publisher:   s.Name,
			Credibility: s.Credibility,
			Excerpt:     s.Description,
			Stance:      stance,
		})
	}

	return &models.FactCheckResult{
		Verdict:         verdict,
		Confidence:      evaluation.Confidence,
		Summary:         evaluation.Summary,
		Claims:          claims,
		Sources:         sources,
		MediaAnalysis:   mediaInfos,
		OriginalContent: *content,
	}
}
