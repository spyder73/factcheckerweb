package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"fact-checker/prompts"
	"fact-checker/services/ai"
)

// AIService provides AI-powered analysis using any configured provider
type AIService struct {
	provider ai.Provider
}

// NewAIService creates a new AI service with the default provider
func NewAIService() (*AIService, error) {
	provider, err := ai.NewDefaultProvider()
	if err != nil {
		return nil, err
	}
	return &AIService{provider: provider}, nil
}

// NewAIServiceWithProvider creates a new AI service with a specific provider
func NewAIServiceWithProvider(provider ai.Provider) *AIService {
	return &AIService{provider: provider}
}

// ProviderName returns the name of the current provider
func (a *AIService) ProviderName() string {
	return a.provider.Name()
}

// ImageAnalysis contains the result of analyzing an image
type ImageAnalysis struct {
	Description string   `json:"description"`
	TextFound   string   `json:"textFound"`
	Elements    []string `json:"elements"`
	Context     string   `json:"context"`
	Claims      []string `json:"claims"`
}

// CondensedInfo contains condensed information from multiple analyses
type CondensedInfo struct {
	MainClaims     []string `json:"mainClaims"`
	KeyFacts       []string `json:"keyFacts"`
	OverallMessage string   `json:"overallMessage"`
	RedFlags       []string `json:"redFlags"`
}

// TruthEvaluation contains the fact-check evaluation
type TruthEvaluation struct {
	Verdict    string       `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Summary    string       `json:"summary"`
	Claims     []ClaimEval  `json:"claims"`
	Sources    []SourceEval `json:"sources"`
}

// ClaimEval contains evaluation of a single claim
type ClaimEval struct {
	Statement       string   `json:"statement"`
	Verdict         string   `json:"verdict"`
	Explanation     string   `json:"explanation"`
	ProArguments    []string `json:"proArguments"`
	ContraArguments []string `json:"contraArguments"`
}

// SourceEval contains information about a source
type SourceEval struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stance      string `json:"stance"`
	Credibility string `json:"credibility"`
}

// AnalyzeImage generates a description of image content
// imageData can be a URL or a base64 data URI
func (a *AIService) AnalyzeImage(imageData string, context string) (ImageAnalysis, error) {
	var response string
	var err error

	if a.provider.SupportsVision() {
		prompt := prompts.Replace(prompts.Prompts.ImageAnalysis, map[string]string{
			"CONTEXT": context,
		})
		response, err = a.provider.AnalyzeImage(imageData, prompt)
	} else {
		// Fallback for non-vision models
		prompt := prompts.Replace(prompts.Prompts.ImageAnalysisFallback, map[string]string{
			"IMAGE_URL": imageData,
			"CONTEXT":   context,
		})
		response, err = a.provider.Chat(prompt)
	}

	if err != nil {
		return ImageAnalysis{}, fmt.Errorf("image analysis failed: %w", err)
	}

	var analysis ImageAnalysis
	if err := json.Unmarshal([]byte(extractJSON(response)), &analysis); err != nil {
		analysis = ImageAnalysis{
			Description: response,
			Elements:    []string{},
			Claims:      []string{},
		}
	}

	return analysis, nil
}

// CondenseInformation combines multiple analyses into key facts
func (a *AIService) CondenseInformation(analyses []ImageAnalysis, caption string) (CondensedInfo, error) {
	var analysisTexts []string
	for i, analysis := range analyses {
		analysisTexts = append(analysisTexts, fmt.Sprintf("Media %d: %s", i+1, analysis.Description))
	}

	systemPrompt := prompts.Prompts.CondenseSystem
	userPrompt := prompts.Replace(prompts.Prompts.CondenseUser, map[string]string{
		"ANALYSES": strings.Join(analysisTexts, "\n\n"),
		"CAPTION":  caption,
	})

	response, err := a.provider.ChatWithSystem(systemPrompt, userPrompt)
	if err != nil {
		return CondensedInfo{}, fmt.Errorf("failed to condense information: %w", err)
	}

	var info CondensedInfo
	if err := json.Unmarshal([]byte(extractJSON(response)), &info); err != nil {
		info = CondensedInfo{
			MainClaims:     []string{caption},
			OverallMessage: response,
		}
	}

	return info, nil
}

// EvaluateTruthfulness performs the main fact-checking
func (a *AIService) EvaluateTruthfulness(info CondensedInfo) (TruthEvaluation, error) {
	systemPrompt := prompts.Prompts.EvaluateSystem
	userPrompt := prompts.Replace(prompts.Prompts.EvaluateUser, map[string]string{
		"MAIN_CLAIMS":     strings.Join(info.MainClaims, "; "),
		"KEY_FACTS":       strings.Join(info.KeyFacts, "; "),
		"OVERALL_MESSAGE": info.OverallMessage,
		"RED_FLAGS":       strings.Join(info.RedFlags, "; "),
	})

	response, err := a.provider.ChatWithSystem(systemPrompt, userPrompt)
	if err != nil {
		return TruthEvaluation{}, fmt.Errorf("failed to evaluate truthfulness: %w", err)
	}

	var eval TruthEvaluation
	if err := json.Unmarshal([]byte(extractJSON(response)), &eval); err != nil {
		eval = TruthEvaluation{
			Verdict:    "unverifiable",
			Confidence: 0.5,
			Summary:    response,
			Claims:     []ClaimEval{},
			Sources:    []SourceEval{},
		}
	}

	return eval, nil
}

// extractJSON attempts to extract JSON from a response that may contain other text
func extractJSON(response string) string {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}
	return response
}
