# FactChecker 🔍

(Quickly Vibe-coded project for fun & testing)

A web application that fact-checks social media posts using AI. Paste any social media link and get an AI-powered analysis of its truthfulness.

![Dark Theme](https://img.shields.io/badge/theme-dark-000000) ![Go](https://img.shields.io/badge/Go-1.21+-00ADD8) ![Python](https://img.shields.io/badge/Python-3.9+-3776AB) ![React](https://img.shields.io/badge/React-18-61DAFB) ![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6)

## ✨ Features

- 🔗 **Multi-platform support** - Instagram, Twitter/X, TikTok, Facebook, YouTube, and more
- 🤖 **AI-powered vision analysis** - Mistral Pixtral analyzes images for claims
- 🖼️ **Parallel image processing** - Multiple images analyzed simultaneously for speed
- 📊 **Real-time progress** - Server-Sent Events (SSE) for live updates
- ✅ **Comprehensive verdicts** - Verified, False, Misleading, Partially True, Unverifiable, Satire, No Claim
- 📝 **Claims breakdown** - Individual analysis of each claim with evidence
- 🔍 **Source references** - Reputable sources cited for verification
- 🎨 **Beautiful dark UI** - Mysterious black/blue/white theme with smooth animations

---

## 🏗️ Architecture

```
┌─────────────┐
│   Frontend  │  React + TypeScript + Tailwind
│  (Port 5173)│  
└──────┬──────┘
       │ HTTP/SSE
       ▼
┌─────────────────────┐
│   Backend (Go)      │  Chi router + concurrent processing
│   (Port 8080)       │  
└──────┬──────────────┘
       │
       ├─► Mistral AI (Vision + Reasoning)
       │   • Pixtral Large (image analysis)
       │   • Mistral Large (fact-checking)
       │
       └─► Instagram Service (Python/Flask)
           • Port 5001
           • Downloads Instagram media
           • Returns base64-encoded images
```

### Component Overview

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | React + TypeScript | User interface with real-time updates |
| **Backend** | Go 1.21+ | Fact-checking orchestration, parallel processing |
| **AI Provider** | Mistral AI | Vision analysis (Pixtral) + reasoning (Large) |
| **Instagram Service** | Python/Flask | Instagram-specific scraping using Instaloader |

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **Python 3.9+** - [Install Python](https://www.python.org/downloads/)
- **Node.js 18+** - [Install Node.js](https://nodejs.org/)
- **Mistral API Key** - [Get one here](https://console.mistral.ai/)

### 1. Start the Instagram Service

```bash
cd instagram-service

# Create virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Run the service
python app.py
```

The Instagram service will be running at `http://localhost:5001`

### 2. Configure and Start the Backend

```bash
cd backend

# Set up environment
cp .env.example .env
# Edit .env and add your MISTRAL_API_KEY

# Install dependencies
go mod tidy

# Run the server
go run main.go
```

The API will be running at `http://localhost:8080`

### 3. Start the Frontend

```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm run dev
```

The app will be running at `http://localhost:5173`

### 4. Open the App

Open your browser and go to **http://localhost:5173**

---

## 📁 Project Structure

```
fact-checker-web/
├── backend/                 # Go API server
│   ├── main.go             # Entry point
│   ├── go.mod              # Go dependencies
│   ├── .env.example        # Environment template
│   ├── config/
│   │   └── config.go       # Environment config loader
│   ├── handlers/
│   │   └── handlers.go     # HTTP handlers (REST + SSE)
│   ├── models/
│   │   └── models.go       # Data structures
│   ├── prompts/            # AI prompt templates
│   │   ├── loader.go       # Template loader (uses embed)
│   │   └── assets/         # Prompt .txt files
│   │       ├── image_analysis.txt
│   │       ├── condense_system.txt
│   │       └── evaluate_system.txt
│   └── services/
│       ├── ai/             # AI providers (modular architecture)
│       │   ├── factory.go  # Provider factory + auto-detection
│       │   ├── types/
│       │   │   └── types.go    # Config + Provider interface
│       │   └── mistral/
│       │       ├── types.go    # Mistral API types
│       │       ├── client.go   # HTTP client
│       │       └── provider.go # Provider implementation
│       ├── scrapers/       # Platform-specific scrapers
│       │   ├── scraper.go  # Scraper interface
│       │   ├── instagram.go    # Instagram (via Python service)
│       │   └── generic.go      # Generic meta-tag scraper
│       ├── aiservice.go    # High-level AI orchestration
│       ├── factcheck.go    # Main workflow engine
│       └── scraper.go      # Scraper registry
│
├── instagram-service/       # Python Flask service
│   ├── app.py              # Flask application
│   ├── requirements.txt    # Python dependencies
│   ├── config.py           # Configuration
│   ├── routes/
│   │   └── instagram_routes.py  # API routes
│   └── utils/
│       └── instagram_utils.py   # Instaloader wrapper
│
└── frontend/               # React app
    ├── package.json
    ├── vite.config.ts
    ├── tailwind.config.js
    └── src/
        ├── App.tsx         # Main component
        ├── index.css       # Global styles
        ├── api/
        │   └── factcheck.ts    # API client
        ├── components/
        │   ├── BackgroundEffects.tsx  # Animated background
        │   ├── LoadingView.tsx        # Progress display
        │   └── ResultView.tsx         # Fact-check results
        └── types/
            └── index.ts    # TypeScript types
```

---

## 🔧 API Endpoints

### Backend (Go) - Port 8080

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/check` | Start a new fact-check |
| `GET` | `/api/check/{id}` | Get check status and result |
| `GET` | `/api/check/{id}/stream` | SSE stream for real-time progress |

### Instagram Service (Python) - Port 5001

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/fetch` | Download Instagram post media |

#### Example: Start a Fact-Check

```bash
curl -X POST http://localhost:8080/api/check \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://www.instagram.com/p/ABC123/",
    "caption": "Optional manual caption"
  }'
```

**Response:**
```json
{
  "id": "uuid-here",
  "status": "processing",
  "progress": 0,
  "currentStep": "Initializing..."
}
```

#### Example: Subscribe to Progress

```javascript
const eventSource = new EventSource(
  'http://localhost:8080/api/check/{id}/stream'
);

eventSource.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(update.step, update.progress, update.message);
};
```

---

## ⚙️ Configuration

### Backend Environment Variables

Create `.env` in `backend/` directory:

```env
# Server Configuration
PORT=8080
HOST=0.0.0.0
DEBUG=false

# AI Provider (required)
AI_PROVIDER=mistral
MISTRAL_API_KEY=your_mistral_api_key_here

# External Services
INSTAGRAM_SERVICE_URL=http://localhost:5001

# Future providers (auto-detected):
# OPENAI_API_KEY=your_openai_key
# ANTHROPIC_API_KEY=your_anthropic_key
```

### Instagram Service Configuration

The Instagram service is configured via `instagram-service/config.py`:

```python
class Config:
    SERVICE_NAME = "instagram-service"
    PORT = 5001
    
    # Instaloader settings
    DOWNLOAD_VIDEOS = False
    DOWNLOAD_VIDEO_THUMBNAILS = False
    DOWNLOAD_GEOTAGS = False
    DOWNLOAD_COMMENTS = False
    SAVE_METADATA = False
```

---

## 🔄 Fact-Checking Workflow

```
1. URL Submission
   ↓
2. Platform Detection
   ↓ (Instagram → Python service, Others → Generic scraper)
3. Content Scraping
   ↓ (Extract caption, author, images as base64)
4. Parallel Image Analysis ⚡
   ↓ (Multiple images analyzed simultaneously)
   ├─► Image 1 → Mistral Pixtral → Analysis 1
   ├─► Image 2 → Mistral Pixtral → Analysis 2
   └─► Image 3 → Mistral Pixtral → Analysis 3
   ↓
5. Information Condensing
   ↓ (Extract key claims from all analyses)
6. Truthfulness Evaluation
   ↓ (Fact-check claims with reasoning)
7. Result Compilation
   ↓ (Build structured response with verdicts + sources)
8. Complete ✓
```

**Performance:** A post with 3 images typically completes in **10-15 seconds** (with parallel processing, images are analyzed in ~2-4s instead of ~6-12s sequentially).

---

## 🎨 Verdict System

| Verdict | Color | Meaning |
|---------|-------|---------|
| ✅ **Verified** | Green | Confirmed as true by reliable sources |
| ❌ **False** | Red | Proven false by evidence |
| ⚠️ **Misleading** | Orange | Contains truth but missing critical context |
| ◐ **Partially True** | Yellow | Some parts true, some false |
| ❓ **Unverifiable** | Gray | Cannot be verified with available sources |
| 😂 **Satire** | Purple | Intentional satire/parody |
| ⊘ **No Claim** | Blue | No factual claims made |

---

## 🧩 Extension Guide

### Adding a New AI Provider

1. **Create provider package:**
   ```
   backend/services/ai/openai/
   ├── types.go      # API types
   ├── client.go     # HTTP client
   └── provider.go   # Provider implementation
   ```

2. **Implement the `types.Provider` interface:**
   ```go
   type Provider interface {
       Name() string
       Chat(message string) (string, error)
       ChatWithSystem(systemPrompt, message string) (string, error)
       AnalyzeImage(imageData string, prompt string) (string, error)
       SupportsVision() bool
   }
   ```

3. **Register in factory** (`services/ai/factory.go`):
   ```go
   case "openai":
       return openai.NewProvider(config), nil
   ```

4. **Add to auto-detection:**
   ```go
   if key := os.Getenv("OPENAI_API_KEY"); key != "" {
       return NewProvider(Config{Provider: "openai", APIKey: key})
   }
   ```

See `backend/README.md` for detailed extension documentation.

### Adding a New Platform Scraper

1. **Create scraper file** (`services/scrapers/tiktok.go`):
   ```go
   func (s *TikTokScraper) CanHandle(url string) bool {
       return strings.Contains(url, "tiktok.com")
   }
   
   func (s *TikTokScraper) Scrape(url string) (*models.ContentInfo, error) {
       // Your implementation
   }
   ```

2. **Register in scraper service** (`services/scraper.go`):
   ```go
   scrapers: []scrapers.Scraper{
       scrapers.NewInstagramScraper(),
       scrapers.NewTikTokScraper(),  // Add here
       scrapers.NewGenericScraper(), // Keep last
   }
   ```

---

## 🧪 Development

### Run Tests

```bash
# Backend tests
cd backend
go test ./...
go test -cover ./...

# Frontend tests
cd frontend
npm test
npm run test:coverage
```

### Build for Production

```bash
# Backend
cd backend
go build -ldflags="-s -w" -o factchecker

# Frontend
cd frontend
npm run build
```

### Docker Support

```dockerfile
# Example Dockerfile for backend
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY backend/ .
RUN go build -o factchecker .

FROM alpine:latest
COPY --from=builder /app/factchecker /factchecker
EXPOSE 8080
CMD ["/factchecker"]
```

---

## 📊 Performance Notes

**Typical Processing Times:**
- Instagram scraping: 1-3s (network + Python service)
- Image analysis: 2-4s per image (parallel, so max time not sum)
- Claim condensing: 2-3s
- Fact evaluation: 3-5s
- **Total**: ~10-15s for a post with 3 images

**Bottlenecks:**
1. Instagram service (network latency)
2. AI vision requests (image size matters)
3. AI reasoning (complex claims take longer)

**Optimizations Applied:**
- ✅ Parallel image analysis (3x speedup for 3 images)
- ✅ Base64 encoding (faster than URL fetching)
- ✅ Goroutines for concurrent processing
- ✅ Connection pooling for HTTP clients

---

## 🔒 Security

- ✅ API keys loaded from environment
- ✅ CORS configured for frontend origin
- ✅ No credentials in version control
- ⚠️ No rate limiting (add for production)
- ⚠️ No authentication (add for production)

**Production Checklist:**
- [ ] Add rate limiting middleware
- [ ] Add API key authentication
- [ ] Validate and sanitize URLs
- [ ] Configure HTTPS
- [ ] Restrict CORS to specific origins
- [ ] Add request timeout handling

---

## 🐛 Troubleshooting

### Instagram Service Not Responding

```bash
# Check if service is running
curl http://localhost:5001/health

# Check logs
cd instagram-service
python app.py  # Run in foreground to see errors
```

### Mistral API Errors

```bash
# Verify API key
echo $MISTRAL_API_KEY

# Test with curl
curl https://api.mistral.ai/v1/chat/completions \
  -H "Authorization: Bearer $MISTRAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"mistral-large-latest","messages":[{"role":"user","content":"test"}]}'
```

### Backend Not Starting

```bash
# Check environment
cd backend
cat .env

# Check logs
go run main.go  # Look for configuration errors
```

---

## 📚 Resources

- **Go Backend Architecture**: See `backend/README.md` for detailed documentation
- **Mistral AI Docs**: https://docs.mistral.ai/
- **Instaloader Docs**: https://instaloader.github.io/
- **React Documentation**: https://react.dev/

---

## 🤝 Contributing

This is a test/demo project, but contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

---

## 📝 License

MIT License - feel free to use this project for learning or as a starting point for your own fact-checker!

---

## ⚡ Future Roadmap

- [ ] Add OpenAI provider (GPT-4 Vision)
- [ ] Add Anthropic provider (Claude 3)
- [ ] Support video content (TikTok, YouTube)
- [ ] Add result caching (Redis)
- [ ] Add user authentication
- [ ] Add historical fact-check database
- [ ] Add browser extension
- [ ] Add API rate limiting
- [ ] Add Prometheus metrics
- [ ] Deploy to production (AWS/GCP)

---

**Built with ❤️ using Go, Python, React, and AI**