package models

import (
	"time"

	"github.com/google/uuid"
)

// CheckRequest represents an incoming fact-check request
type CheckRequest struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

// CheckResponse represents the fact-check result
type CheckResponse struct {
	ID             string           `json:"id"`
	Status         CheckStatus      `json:"status"`
	Progress       int              `json:"progress"`
	CurrentStep    string           `json:"currentStep"`
	Result         *FactCheckResult `json:"result,omitempty"`
	Error          string           `json:"error,omitempty"`
	ProcessingTime float64          `json:"processingTime"`
	CreatedAt      time.Time        `json:"createdAt"`
}

type CheckStatus string

const (
	StatusPending    CheckStatus = "pending"
	StatusProcessing CheckStatus = "processing"
	StatusCompleted  CheckStatus = "completed"
	StatusError      CheckStatus = "error"
)

// FactCheckResult contains the complete analysis
type FactCheckResult struct {
	Verdict         Verdict     `json:"verdict"`
	Confidence      float64     `json:"confidence"`
	Summary         string      `json:"summary"`
	Claims          []Claim     `json:"claims"`
	Sources         []Source    `json:"sources"`
	MediaAnalysis   []MediaInfo `json:"mediaAnalysis"`
	OriginalContent ContentInfo `json:"originalContent"`
}

type Verdict string

const (
	VerdictVerified     Verdict = "verified"
	VerdictFalse        Verdict = "false"
	VerdictMisleading   Verdict = "misleading"
	VerdictPartialTrue  Verdict = "partially_true"
	VerdictUnverifiable Verdict = "unverifiable"
	VerdictSatire       Verdict = "satire"
	VerdictNoClaim      Verdict = "no_claim"
)

// Claim represents a specific claim found in the content
type Claim struct {
	ID              string   `json:"id"`
	Statement       string   `json:"statement"`
	Verdict         Verdict  `json:"verdict"`
	Explanation     string   `json:"explanation"`
	ProArguments    []string `json:"proArguments"`
	ContraArguments []string `json:"contraArguments"`
	SourceIDs       []string `json:"sourceIds"`
}

// Source represents a reference source
type Source struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url,omitempty"`
	Publisher   string     `json:"publisher"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	Credibility string     `json:"credibility"`
	Excerpt     string     `json:"excerpt,omitempty"`
	Stance      Stance     `json:"stance"`
}

type Stance string

const (
	StanceSupports    Stance = "supports"
	StanceContradicts Stance = "contradicts"
	StanceNeutral     Stance = "neutral"
	StanceContext     Stance = "context"
)

// MediaInfo contains analysis of a single media item
type MediaInfo struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "image" or "video_frame"
	URL         string   `json:"url,omitempty"`
	Description string   `json:"description"`
	Elements    []string `json:"elements"`
	TextFound   string   `json:"textFound,omitempty"`
}

// ContentInfo contains the original post information
type ContentInfo struct {
	Platform  string   `json:"platform"`
	URL       string   `json:"url"`
	Caption   string   `json:"caption"`
	MediaURLs []string `json:"mediaUrls"`
	Author    string   `json:"author,omitempty"`
	PostedAt  string   `json:"postedAt,omitempty"`
}

// ProgressUpdate for SSE streaming
type ProgressUpdate struct {
	Step     string `json:"step"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

// NewCheckResponse creates a new check response
func NewCheckResponse() *CheckResponse {
	return &CheckResponse{
		ID:        uuid.New().String(),
		Status:    StatusPending,
		Progress:  0,
		CreatedAt: time.Now(),
	}
}
