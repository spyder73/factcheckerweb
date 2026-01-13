package services

import (
	"context"
	"fmt"
	"log"
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

// mediaAnalysisResult holds the result of analyzing a single media item
type mediaAnalysisResult struct {
	index    int
	analysis ImageAnalysis
	info     models.MediaInfo
	err      error
}

// analyzeMediaParallel processes multiple media items concurrently
func (f *FactCheckService) analyzeMediaParallel(id string, mediaURLs []string, caption string) ([]ImageAnalysis, []models.MediaInfo, error) {
	if len(mediaURLs) == 0 {
		return nil, nil, nil
	}

	log.Printf("[Fact Checker] Starting parallel analysis of %d media items", len(mediaURLs))
	startTime := time.Now()

	// Create channels for results and progress
	resultChan := make(chan mediaAnalysisResult, len(mediaURLs))
	var wg sync.WaitGroup

	// Launch goroutine for each media item
	for i, mediaURL := range mediaURLs {
		wg.Add(1)
		go func(index int, url string) {
			defer wg.Done()

			log.Printf("[Fact Checker] Analyzing media %d/%d (goroutine %d)", index+1, len(mediaURLs), index)

			// Perform analysis
			analysis, err := f.ai.AnalyzeImage(url, caption)

			result := mediaAnalysisResult{
				index: index,
				err:   err,
			}

			if err == nil {
				result.analysis = analysis
				result.info = models.MediaInfo{
					ID:          uuid.New().String(),
					Type:        "image",
					URL:         url,
					Description: analysis.Description,
					Elements:    analysis.Elements,
					TextFound:   analysis.TextFound,
				}
				log.Printf("[Fact Checker] Completed analysis of media %d/%d", index+1, len(mediaURLs))
			} else {
				log.Printf("[Fact Checker] Failed to analyze media %d/%d: %v", index+1, len(mediaURLs), err)
			}

			resultChan <- result

			// Update progress (approximate)
			progressIncrement := 25 / len(mediaURLs)
			currentProgress := 25 + ((index + 1) * progressIncrement)
			f.updateProgress(id, "analyzing_media", currentProgress,
				fmt.Sprintf("Analyzed %d/%d images", index+1, len(mediaURLs)))
		}(i, mediaURL)
	}

	// Wait for all analyses to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	results := make([]mediaAnalysisResult, len(mediaURLs))
	successCount := 0
	errorCount := 0

	for result := range resultChan {
		results[result.index] = result
		if result.err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	log.Printf("[Fact Checker] Parallel analysis completed in %v - Success: %d, Errors: %d",
		time.Since(startTime), successCount, errorCount)

	// Extract successful analyses in original order
	var mediaAnalyses []ImageAnalysis
	var mediaInfos []models.MediaInfo

	for _, result := range results {
		if result.err == nil {
			mediaAnalyses = append(mediaAnalyses, result.analysis)
			mediaInfos = append(mediaInfos, result.info)
		}
	}

	// Return error only if ALL analyses failed
	if len(mediaAnalyses) == 0 && len(mediaURLs) > 0 {
		return nil, nil, fmt.Errorf("all media analyses failed")
	}

	return mediaAnalyses, mediaInfos, nil
}

func (f *FactCheckService) processCheck(id string, req models.CheckRequest) {
	startTime := time.Now()
	ctx := context.Background()
	_ = ctx // For future cancellation support

	log.Printf("[Fact Checker] Starting fact-check %s for URL: %s", id, req.URL)

	// Step 1: Scrape content
	f.updateProgress(id, "scraping", 10, "Fetching content from URL...")

	content, err := f.scraper.ScrapePost(req.URL)
	if err != nil {
		log.Printf("[Fact Checker] Scraping failed for %s: %v", id, err)
		f.setError(id, "Failed to fetch content: "+err.Error())
		return
	}

	log.Printf("[Fact Checker] Scraped content for %s - Platform: %s, Media items: %d, Caption length: %d",
		id, content.Platform, len(content.MediaURLs), len(content.Caption))

	// Add manual caption if provided
	if req.Caption != "" && content.Caption == "" {
		content.Caption = req.Caption
		log.Printf("[Fact Checker] Using manual caption for %s", id)
	}

	// Step 2: Analyze media (PARALLEL)
	f.updateProgress(id, "analyzing_media", 25, "Analyzing images in parallel...")

	mediaAnalyses, mediaInfos, err := f.analyzeMediaParallel(id, content.MediaURLs, content.Caption)
	if err != nil {
		log.Printf("[Fact Checker] Media analysis failed for %s: %v", id, err)
		f.setError(id, "Failed to analyze media: "+err.Error())
		return
	}

	// If no media was analyzed successfully, try to work with caption only
	if len(mediaAnalyses) == 0 && content.Caption != "" {
		log.Printf("[Fact Checker] No media analyzed for %s, using caption only", id)
		mediaAnalyses = append(mediaAnalyses, ImageAnalysis{
			Description: content.Caption,
			Claims:      []string{content.Caption},
		})
	}

	// If still no content, fail
	if len(mediaAnalyses) == 0 {
		log.Printf("[Fact Checker] No content to analyze for %s", id)
		f.setError(id, "No analyzable content found")
		return
	}

	log.Printf("[Fact Checker] Completed media analysis for %s - Analyzed %d items", id, len(mediaAnalyses))

	// Step 3: Condense information
	f.updateProgress(id, "condensing", 50, "Extracting key claims...")

	condensed, err := f.ai.CondenseInformation(mediaAnalyses, content.Caption)
	if err != nil {
		log.Printf("[Fact Checker] Information condensing failed for %s: %v", id, err)
		f.setError(id, "Failed to analyze content: "+err.Error())
		return
	}

	log.Printf("[Fact Checker] Condensed information for %s - Claims: %d, Red flags: %d",
		id, len(condensed.MainClaims), len(condensed.RedFlags))

	// Step 4: Evaluate truthfulness
	f.updateProgress(id, "evaluating", 70, "Verifying claims against sources...")

	evaluation, err := f.ai.EvaluateTruthfulness(condensed)
	if err != nil {
		log.Printf("[Fact Checker] Truthfulness evaluation failed for %s: %v", id, err)
		f.setError(id, "Failed to evaluate claims: "+err.Error())
		return
	}

	log.Printf("[Fact Checker] Evaluation complete for %s - Verdict: %s, Confidence: %.2f, Claims: %d, Sources: %d",
		id, evaluation.Verdict, evaluation.Confidence, len(evaluation.Claims), len(evaluation.Sources))

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

	log.Printf("[Fact Checker] Fact-check %s completed in %.2fs", id, time.Since(startTime).Seconds())

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
