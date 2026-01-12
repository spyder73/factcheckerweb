# FactChecker 🔍

A web application that fact-checks social media posts using AI. Paste any social media link and get an AI-powered analysis of its truthfulness.

![Dark Theme](https://img.shields.io/badge/theme-dark-000000) ![Go](https://img.shields.io/badge/Go-1.21+-00ADD8) ![React](https://img.shields.io/badge/React-18-61DAFB) ![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6)

## Features

- 🔗 **Paste any social media link** - Supports Instagram, Twitter/X, TikTok, Facebook, YouTube
- 🤖 **AI-powered analysis** - Uses Duck.ai for content analysis
- 📊 **Real-time progress** - Watch the fact-checking process live
- ✅ **Verdict system** - Verified, False, Misleading, Partially True, Unverifiable, Satire
- 📝 **Claims breakdown** - Individual analysis of each claim
- 🔍 **Source references** - See what sources were used
- 🎨 **Beautiful dark UI** - Mysterious black/blue/white theme

---

## Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **Node.js 18+** - [Install Node.js](https://nodejs.org/)
- **Mistral API Key** - [Get one here](https://console.mistral.ai/)

### 1. Configure Environment

```bash
cd backend
cp .env.example .env
# Edit .env and add your MISTRAL_API_KEY
```

### 2. Start the Backend

```bash
cd backend
go mod tidy
go run main.go
```

The API will be running at `http://localhost:8080`

### 3. Start the Frontend (in a new terminal)

```bash
cd frontend
npm install
npm run dev
```

The app will be running at `http://localhost:5173`

### 4. Open the App

Open your browser and go to **http://localhost:5173**

---

## Project Structure

```
fact-checker-web/
├── backend/                 # Go API server
│   ├── main.go             # Entry point
│   ├── go.mod              # Go dependencies
│   ├── .env.example        # Environment template
│   ├── handlers/           # HTTP handlers
│   │   └── handlers.go
│   ├── models/             # Data models
│   │   └── models.go
│   └── services/           # Business logic
│       ├── ai/             # AI providers (modular)
│       │   ├── provider.go # Provider interface
│       │   ├── factory.go  # Provider factory
│       │   └── mistral.go  # Mistral implementation
│       ├── aiservice.go    # AI service wrapper
│       ├── factcheck.go    # Fact-check orchestration
│       └── scraper.go      # URL scraping
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
        │   ├── BackgroundEffects.tsx
        │   ├── LoadingView.tsx
        │   └── ResultView.tsx
        └── types/
            └── index.ts    # TypeScript types
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/check` | Start a fact-check |
| `GET` | `/api/check/{id}` | Get check status |
| `GET` | `/api/check/{id}/stream` | SSE progress stream |

### Example: Start a Fact-Check

```bash
curl -X POST http://localhost:8080/api/check \
  -H "Content-Type: application/json" \
  -d '{"url": "https://twitter.com/example/status/123"}'
```

---

## Environment Variables

Create a `.env` file in the `backend/` directory:

```env
PORT=8080

# Required: Mistral AI
MISTRAL_API_KEY=your_key_here

# Future providers (the system auto-detects which key is available):
# OPENAI_API_KEY=your_key
# ANTHROPIC_API_KEY=your_key
```

---

## Adding New AI Providers

The AI service is modular. To add a new provider:

1. Create `backend/services/ai/newprovider.go` implementing the `Provider` interface
2. Add a case in `backend/services/ai/factory.go`
3. Set the corresponding API key in `.env`

```go
// Example: backend/services/ai/provider.go
type Provider interface {
    Name() string
    Chat(message string) (string, error)
    ChatWithSystem(systemPrompt, message string) (string, error)
    AnalyzeImage(imageURL string, prompt string) (string, error)
    SupportsVision() bool
}
```

---

## Tech Stack

**Backend:**
- Go 1.21+
- Chi router
- Mistral AI (modular, expandable to OpenAI, Anthropic, etc.)

**Frontend:**
- React 18
- TypeScript
- Vite
- Tailwind CSS
- Framer Motion
- Lucide Icons

---

## License

MIT
