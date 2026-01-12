package main

import (
	"log"
	"net/http"
	"os"

	"fact-checker/handlers"
	"fact-checker/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Initialize AI service (uses MISTRAL_API_KEY from environment)
	aiService, err := services.NewAIService()
	if err != nil {
		log.Fatalf("❌ Failed to initialize AI service: %v", err)
	}
	log.Printf("✅ AI Provider: %s", aiService.ProviderName())

	// Initialize other services
	scraper := services.NewScraperService()
	factChecker := services.NewFactCheckService(aiService, scraper)

	// Initialize handlers
	h := handlers.NewHandler(factChecker)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/health", h.HealthCheck)
	r.Post("/api/check", h.CheckFacts)
	r.Get("/api/check/{id}", h.GetCheckStatus)
	r.Get("/api/check/{id}/stream", h.StreamCheckProgress)

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 FactChecker API running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
