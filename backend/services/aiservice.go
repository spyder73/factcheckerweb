package services

import (
	"encoding/json"
	"fmt"
	"strings"

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
func (a *AIService) AnalyzeImage(imageURL string, context string) (ImageAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze this image from a social media post.

Additional context/caption: %s

Please provide:
1. A detailed description of what is visible in the image
2. Any text that appears in the image (transcribe exactly)
3. Key elements and objects visible
4. The apparent context or setting
5. Any claims or statements being made visually

Be objective and factual. Format your response as JSON:
{
  "description": "detailed description",
  "textFound": "any text in image",
  "elements": ["element1", "element2"],
  "context": "apparent context",
  "claims": ["claim1", "claim2"]
}`, context)

	var response string
	var err error

	// Use vision if supported, otherwise describe what we're analyzing
	if a.provider.SupportsVision() {
		response, err = a.provider.AnalyzeImage(imageURL, prompt)
	} else {
		// Fallback for non-vision models
		textPrompt := fmt.Sprintf(`I need you to analyze a social media post. 
The post contains an image at this URL: %s
Caption/context: %s

Since you cannot see the image directly, analyze the caption and URL for any claims that can be fact-checked.
Return your analysis as JSON:
{
  "description": "analysis of available text content",
  "textFound": "",
  "elements": [],
  "context": "context from caption",
  "claims": ["claims found in caption"]
}`, imageURL, context)
		response, err = a.provider.Chat(textPrompt)
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

	systemPrompt := `You are a fact-checking assistant. Your job is to extract verifiable claims from social media content. Be precise and identify specific factual claims that can be checked.`

	prompt := fmt.Sprintf(`Given the following analyses of social media content, condense them into key verifiable claims.

Analyses:
%s

Original Caption: %s

Extract and return as JSON:
{
  "mainClaims": ["primary claim 1", "primary claim 2"],
  "keyFacts": ["fact that can be verified"],
  "overallMessage": "the main message being conveyed",
  "redFlags": ["any potential misinformation indicators"]
}`, strings.Join(analysisTexts, "\n\n"), caption)

	response, err := a.provider.ChatWithSystem(systemPrompt, prompt)
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
	systemPrompt := `You are an expert fact-checker. Evaluate claims objectively based on known facts, reputable sources, and logical analysis. Always cite your reasoning and acknowledge uncertainty when appropriate.`

	prompt := fmt.Sprintf(`Perform a comprehensive fact-check on the following claims from social media content.

Main Claims: %s

Key Facts Presented: %s

Overall Message: %s

Red Flags Identified: %s

Please analyze and provide:
1. VERIFICATION STATUS for each claim (verified, false, misleading, partially_true, unverifiable, satire, no_claim)
2. SUPPORTING EVIDENCE: Known facts that support the claims
3. CONTRADICTING EVIDENCE: Known facts that contradict the claims
4. SOURCES: Mention reputable news organizations, fact-checkers, or official sources
5. MISSING CONTEXT: Important information that changes the meaning
6. OVERALL VERDICT: Your assessment

Return as JSON:
{
  "verdict": "verified|false|misleading|partially_true|unverifiable|satire|no_claim",
  "confidence": 0.0-1.0,
  "summary": "overall summary explaining the verdict",
  "claims": [
    {
      "statement": "the claim",
      "verdict": "verdict for this claim",
      "explanation": "detailed explanation why",
      "proArguments": ["supporting points"],
      "contraArguments": ["contradicting points"]
    }
  ],
  "sources": [
    {
      "name": "source name (e.g., Reuters, AP News, Snopes)",
      "description": "what they reported or would report",
      "stance": "supports|contradicts|neutral|context",
      "credibility": "high|medium|low"
    }
  ]
}`,
		strings.Join(info.MainClaims, "; "),
		strings.Join(info.KeyFacts, "; "),
		info.OverallMessage,
		strings.Join(info.RedFlags, "; "))

	response, err := a.provider.ChatWithSystem(systemPrompt, prompt)
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
