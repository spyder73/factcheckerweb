# FactChecker Backend

A modular, extensible Go backend for fact-checking social media content using AI.

## 📁 Architecture Overview

```
backend/
├── main.go                     # Application entry point
├── config/
│   └── config.go               # Environment configuration loader
├── handlers/
│   └── handlers.go             # HTTP request handlers
├── models/
│   └── models.go               # Data structures & types
├── prompts/
│   ├── loader.go               # Prompt template loader (uses embed)
│   └── assets/                 # Prompt template files
│       ├── image_analysis.txt
│       ├── image_analysis_fallback.txt
│       ├── condense_system.txt
│       ├── condense_user.txt
│       ├── evaluate_system.txt
│       └── evaluate_user.txt
└── services/
    ├── aiservice.go            # High-level AI orchestration
    ├── factcheck.go            # Fact-check workflow engine
    ├── scraper.go              # Scraper registry/router
    ├── ai/
    │   ├── factory.go          # Provider factory + re-exports
    │   ├── types/
    │   │   └── types.go        # Config + Provider interface
    │   └── mistral/
    │       ├── types.go        # Mistral API types
    │       ├── client.go       # HTTP client for Mistral
    │       └── provider.go     # Mistral provider implementation
    └── scrapers/
        ├── scraper.go          # Scraper interface
        ├── instagram.go        # Instagram scraper (via Python service)
        └── generic.go          # Generic meta-tag scraper
```

---

## 🔧 Core Components

### 1. Configuration (`config/`)

Centralized configuration management using environment variables.

```go
import "fact-checker/config"

func main() {
    config.Load()
    fmt.Println(config.App.Port)
}
```

**Environment Variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `HOST` | `0.0.0.0` | Server host |
| `MISTRAL_API_KEY` | - | Mistral AI API key (required) |
| `INSTAGRAM_SERVICE_URL` | `http://localhost:5001` | Instagram service URL |
| `DEBUG` | `false` | Enable debug logging |

---

### 2. HTTP Handlers (`handlers/`)

Chi-based HTTP handlers for the REST API.

**Endpoints:**
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/check` | Start fact-check |
| `GET` | `/api/check/{id}` | Get check status |
| `GET` | `/api/check/{id}/stream` | SSE progress stream |

---

### 3. Models (`models/`)

Data structures used throughout the application.

**Key Types:**
- `CheckRequest` - Incoming fact-check request
- `CheckResponse` - Full response with status & result
- `FactCheckResult` - Complete analysis result
- `Claim` - Individual claim with verdict
- `ContentInfo` - Scraped content data

---

### 4. Prompt Templates (`prompts/`)

External prompt files loaded at compile time using Go's `embed` package.

**Structure:**
```
prompts/
├── loader.go          # Template loader with Replace() utility
└── assets/            # All .txt prompt files
    ├── image_analysis.txt
    ├── condense_system.txt
    └── evaluate_system.txt
```

**Why external files?**
- ✅ Easier to edit and version control
- ✅ No code changes needed for prompt tweaks
- ✅ Clear separation of concerns
- ✅ Single binary deployment (embedded at compile time)

**Usage:**
```go
import "fact-checker/prompts"

prompt := prompts.Replace(prompts.Prompts.ImageAnalysis, map[string]string{
    "CONTEXT": "Some caption text",
})
```

---

### 5. Services (`services/`)

#### 5.1 Fact-Check Service (`factcheck.go`)

The main orchestrator that coordinates the entire fact-checking workflow:

```
URL Input
    │
    ▼
┌─────────────────┐
│  1. Scrape      │ ──► Platform-specific scraper
└─────────────────┘     (Instagram, Generic, etc.)
    │
    ▼
┌─────────────────┐
│  2. Analyze     │ ──► AI vision for each image
└─────────────────┘     (Base64 data URIs → Mistral Pixtral)
    │
    ▼
┌─────────────────┐
│  3. Condense    │ ──► Extract key claims from analyses
└─────────────────┘     (Multiple images → Single claim list)
    │
    ▼
┌─────────────────┐
│  4. Evaluate    │ ──► Fact-check claims with reasoning
└─────────────────┘     (Claims → Verdicts + Sources)
    │
    ▼
┌─────────────────┐
│  5. Build       │ ──► Format final result as JSON
└─────────────────┘
```

**Progress Updates:**
The service emits real-time progress via Server-Sent Events (SSE):
- `scraping` - Fetching content from platform
- `analyzing` - Processing images with AI
- `condensing` - Extracting claims
- `evaluating` - Fact-checking claims
- `completed` - Result ready

#### 5.2 AI Service (`aiservice.go`)

High-level AI operations that use prompt templates:

- **`AnalyzeImage(imageData, context)`** - Vision analysis of images
  - Accepts base64 data URIs from scrapers
  - Uses `image_analysis.txt` prompt template
  - Returns structured JSON with claims, text, elements
  
- **`CondenseInformation(analyses, caption)`** - Extract claims from analyses
  - Combines multiple image analyses
  - Uses `condense_system.txt` + `condense_user.txt`
  - Returns main claims, key facts, red flags
  
- **`EvaluateTruthfulness(info)`** - Perform fact-checking
  - Takes condensed information
  - Uses `evaluate_system.txt` + `evaluate_user.txt`
  - Returns verdict, confidence, sources, claim breakdown

#### 5.3 Scraper Service (`scraper.go`)

Routes URLs to appropriate platform scrapers using a **chain of responsibility** pattern.

**Scraper Priority:**
1. `InstagramScraper` - Handles Instagram URLs via Python service
2. `GenericScraper` - Fallback for all other platforms (meta tags)

---

## 🔌 Extension Points

### Adding a New AI Provider

1. **Create provider package:**

```
services/ai/openai/
├── types.go      # OpenAI API request/response types
├── client.go     # HTTP client for OpenAI API
└── provider.go   # Provider implementation
```

2. **Implement the `types.Provider` interface:**

```go
// services/ai/openai/provider.go
package openai

import "fact-checker/services/ai/types"

type Provider struct {
    client *Client
    model  string
}

func NewProvider(config types.Config) *Provider {
    return &Provider{
        client: NewClient(config.APIKey, config.BaseURL),
        model:  config.Model,
    }
}

func (p *Provider) Name() string { return "openai" }
func (p *Provider) SupportsVision() bool { return true }
func (p *Provider) Chat(message string) (string, error) { /* ... */ }
func (p *Provider) ChatWithSystem(sys, msg string) (string, error) { /* ... */ }
func (p *Provider) AnalyzeImage(img, prompt string) (string, error) { /* ... */ }
```

3. **Register in factory:**

```go
// services/ai/factory.go
func NewProvider(config Config) (Provider, error) {
    switch config.Provider {
    case "mistral":
        return mistral.NewProvider(config), nil
    case "openai":  // ← Add this
        if config.APIKey == "" {
            config.APIKey = os.Getenv("OPENAI_API_KEY")
        }
        return openai.NewProvider(config), nil
    // ...
    }
}
```

4. **Add to auto-detection:**

```go
func NewDefaultProvider() (Provider, error) {
    if key := os.Getenv("MISTRAL_API_KEY"); key != "" {
        return NewProvider(Config{Provider: "mistral", APIKey: key})
    }
    if key := os.Getenv("OPENAI_API_KEY"); key != "" {  // ← Add this
        log.Printf("[AI Factory] Found OPENAI_API_KEY, using OpenAI")
        return NewProvider(Config{Provider: "openai", APIKey: key})
    }
    // ...
}
```

**That's it!** The system will automatically use OpenAI if `OPENAI_API_KEY` is set.

---

### Adding a New Social Media Platform

1. **Create scraper file:**

```go
// services/scrapers/tiktok.go
package scrapers

import (
    "fact-checker/models"
    "net/http"
    "strings"
    "time"
)

type TikTokScraper struct {
    client *http.Client
}

func NewTikTokScraper() *TikTokScraper {
    return &TikTokScraper{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

func (s *TikTokScraper) Platform() string {
    return "tiktok"
}

func (s *TikTokScraper) CanHandle(url string) bool {
    return strings.Contains(strings.ToLower(url), "tiktok.com")
}

func (s *TikTokScraper) Scrape(url string) (*models.ContentInfo, error) {
    // Your implementation:
    // 1. Fetch TikTok video page
    // 2. Extract video frames or thumbnail
    // 3. Extract caption, author
    // 4. Return ContentInfo with base64 images
    
    return &models.ContentInfo{
        Platform:  "tiktok",
        URL:       url,
        Caption:   "extracted caption",
        Author:    "@username",
        MediaURLs: []string{"data:image/jpeg;base64,..."},
    }, nil
}
```

2. **Register in scraper service:**

```go
// services/scraper.go
func NewScraperService() *ScraperService {
    return &ScraperService{
        scrapers: []scrapers.Scraper{
            scrapers.NewInstagramScraper(),
            scrapers.NewTikTokScraper(),  // ← Add here (order matters!)
            scrapers.NewGenericScraper(), // Keep last as fallback
        },
    }
}
```

**Priority matters!** Scrapers are checked in order. Put specific platforms before generic.

---

### Adding New Prompt Templates

1. **Create prompt file:**

```text
// prompts/assets/claim_verification.txt
You are verifying a specific claim from social media.

Claim: {{CLAIM}}
Context: {{CONTEXT}}

Please research this claim and provide:
1. Is this claim TRUE, FALSE, or UNVERIFIABLE?
2. What evidence supports or contradicts it?
3. What reputable sources have reported on this?

Format as JSON:
{
  "verdict": "true|false|unverifiable",
  "confidence": 0.0-1.0,
  "evidence": ["fact 1", "fact 2"],
  "sources": ["Reuters: ...", "AP: ..."]
}
```

2. **Add to loader:**

```go
// prompts/loader.go
type PromptTemplates struct {
    // ...existing prompts...
    ClaimVerification string  // ← Add this
}

func loadPrompts() PromptTemplates {
    return PromptTemplates{
        // ...existing prompts...
        ClaimVerification: mustRead("assets/claim_verification.txt"),  // ← Add this
    }
}
```

3. **Use in code:**

```go
prompt := prompts.Replace(prompts.Prompts.ClaimVerification, map[string]string{
    "CLAIM": "The moon landing was fake",
    "CONTEXT": "Posted on Instagram with old photos",
})
result, err := aiService.provider.Chat(prompt)
```

**Note:** Prompts are embedded at compile time, so you need to rebuild after adding new templates.

---

### Adding New API Endpoints

1. **Add handler method:**

```go
// handlers/handlers.go
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
    stats := map[string]interface{}{
        "totalChecks": len(h.factChecker.checks),
        "completed":   /* count completed */,
        "pending":     /* count pending */,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}
```

2. **Register route in `main.go`:**

```go
// main.go
r.Get("/api/stats", handler.GetStatistics)  // ← Add this line
```

**CORS:** Already configured to allow all origins in `main.go`.

---

## 🏗️ Design Principles

### 1. Interface-Driven Design
All major components implement interfaces, making them easily swappable:
- **`types.Provider`** - AI providers (Mistral, OpenAI, Anthropic, etc.)
- **`scrapers.Scraper`** - Platform scrapers (Instagram, TikTok, etc.)

**Benefits:**
- Easy to mock for testing
- Swap implementations without code changes
- Add new providers/scrapers without modifying existing code

### 2. Factory Pattern
Providers are created through factories that handle:
- ✅ Configuration validation
- ✅ Environment variable loading
- ✅ Default value assignment
- ✅ Auto-detection of available providers

### 3. Separation of Concerns
Clear boundaries between layers:
| Layer | Responsibility | No Access To |
|-------|----------------|--------------|
| **Handlers** | HTTP requests/responses | Business logic |
| **Services** | Business logic & orchestration | HTTP details |
| **Models** | Data structures only | Logic or I/O |
| **Config** | Environment management | Business logic |

### 4. Embedded Resources
Prompt templates are embedded at compile time using `//go:embed`, ensuring:
- ✅ Single binary deployment
- ✅ No runtime file dependencies
- ✅ Version-controlled prompts
- ✅ Fast access (no disk I/O)

### 5. Extensive Logging
All major operations log with consistent format: `[Component] Message`

**What's logged:**
- Request details (sanitized, no sensitive data)
- Timing information (operation duration)
- Response summaries (token counts, size)
- Errors with full context
- Progress milestones

**Example:**
```
[Instagram Scraper] Starting scrape for URL: https://instagram.com/p/ABC123
[Instagram Scraper] Received response (status: 200, size: 45632 bytes)
[Mistral] Sending vision request to model: pixtral-large-latest
[Mistral] Vision response received in 2.3s (tokens: prompt=1234, completion=567)
```

### 6. Async Processing with SSE
Fact-checking happens asynchronously with real-time progress updates:
- Client submits URL → Gets check ID immediately
- Server processes in background goroutine
- Client subscribes to SSE stream for live updates
- Each step emits progress: `scraping` → `analyzing` → `evaluating` → `completed`

---

## 🚀 Running the Backend

```bash
# 1. Set up environment
cp .env.example .env
# Edit .env and add your MISTRAL_API_KEY

# 2. Install dependencies
go mod tidy

# 3. Run development server
go run main.go

# The server will start on http://localhost:8080
```

**Production build:**
```bash
# Build optimized binary
go build -ldflags="-s -w" -o factchecker main.go

# Run
./factchecker
```

**Docker:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o factchecker .

FROM alpine:latest
COPY --from=builder /app/factchecker /factchecker
EXPOSE 8080
CMD ["/factchecker"]
```

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./services/...

# Run with verbose output
go test -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Testing tips:**
- Mock the `types.Provider` interface for AI service tests
- Mock the `scrapers.Scraper` interface for scraper tests
- Use `httptest` for handler tests

---

## 📝 Logging Format

All logs follow the pattern: `[Component] Message`

**Components:**
- `[Config]` - Configuration loading
- `[AI Factory]` - Provider creation
- `[Mistral]` - Mistral API calls
- `[Instagram Scraper]` - Instagram scraping
- `[Fact Checker]` - Main workflow
- `[Handler]` - HTTP requests

**Example output:**
```
2024/01/13 15:23:45 [Config] Loaded configuration:
2024/01/13 15:23:45 [Config]   Port: 8080
2024/01/13 15:23:45 [Config]   AI Provider: mistral
2024/01/13 15:23:45 [Config]   Instagram Service: http://localhost:5001
2024/01/13 15:23:50 [Instagram Scraper] Starting scrape for URL: https://instagram.com/p/ABC
2024/01/13 15:23:52 [Instagram Scraper] Scrape completed in 2.1s - Images: 3
2024/01/13 15:23:52 [Mistral] Sending vision request to model: pixtral-large-latest
2024/01/13 15:23:55 [Mistral] Vision response received in 2.8s (tokens: 1523)
```

---

## 🔮 Future Improvements

### Short Term
- [ ] Add request rate limiting per IP
- [ ] Add result caching (Redis)
- [ ] Add request ID tracing
- [ ] Add health check for Instagram service
- [ ] Add graceful shutdown

### Medium Term
- [ ] Add OpenAI provider (GPT-4 Vision)
- [ ] Add Anthropic provider (Claude 3)
- [ ] Add video frame extraction for TikTok/YouTube
- [ ] Add unit tests for all services
- [ ] Add integration tests

### Long Term
- [ ] Add PostgreSQL for result persistence
- [ ] Add user authentication & API keys
- [ ] Add Prometheus metrics
- [ ] Add OpenTelemetry tracing
- [ ] Add admin dashboard
- [ ] Add webhook notifications

---

## 📊 Performance Notes

**Typical Request Times:**
- Scraping: 1-3 seconds (depends on platform)
- Image analysis: 2-4 seconds per image (Mistral Pixtral)
- Fact evaluation: 3-5 seconds (Mistral Large)
- **Total**: ~10-15 seconds for a post with 2-3 images

**Bottlenecks:**
1. Instagram service (network + Python processing)
2. AI vision requests (image size matters)
3. AI reasoning (complex claims take longer)

**Optimization Tips:**
- Process images in parallel (currently sequential)
- Cache common Instagram posts
- Use smaller image sizes (resize before sending)
- Implement request pooling for AI calls

---

## 🔒 Security Considerations

- ✅ API keys loaded from environment (not hardcoded)
- ✅ No SQL injection risk (no database yet)
- ✅ CORS configured for frontend
- ⚠️ No rate limiting (add for production)
- ⚠️ No authentication (add for production)
- ⚠️ No input validation on URLs (add sanitization)

**Production checklist:**
- [ ] Add rate limiting middleware
- [ ] Add API key authentication
- [ ] Validate and sanitize all inputs
- [ ] Add HTTPS support
- [ ] Configure CORS for specific origins only
- [ ] Add request size limits
- [ ] Add timeout handling for all external calls

---

## 📄 License

MIT