package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alethea/api/audit"
	"alethea/api/auth"
	"alethea/api/config"
	"alethea/api/db"
	"alethea/api/handlers"
	"alethea/api/httpx"
	alog "alethea/api/log"
	"alethea/api/ratelimit"
	"alethea/api/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// godotenv.Load is best-effort for the file-missing case (we read live
	// env in prod), but parse errors on a present .env are unambiguous bugs.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		// slog isn't initialized yet — use stdlib briefly.
		log.Printf("dotenv parse error: %v", err)
		os.Exit(1)
	}

	// Logging first so anything else logs through our redactor.
	alog.Init(os.Getenv("DEBUG") == "true")

	if err := config.Load(); err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	cfg := config.App

	// Trusted-proxy registry MUST be set before any handler runs ClientIP.
	// Empty list = never honor X-Forwarded-For (correct default for direct exposure).
	prefixes, err := httpx.ParseProxyCIDRs(cfg.TrustedProxies)
	if err != nil {
		slog.Warn("TRUSTED_PROXIES contained bad entries — those entries ignored", "err", err)
	}
	httpx.SetTrustedProxies(prefixes)
	if len(prefixes) == 0 {
		slog.Info("TRUSTED_PROXIES is empty — X-Forwarded-For will be ignored (use r.RemoteAddr)")
	} else {
		slog.Info("trusted proxies configured", "count", len(prefixes))
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Postgres
	pool, err := db.Open(rootCtx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(rootCtx, pool); err != nil {
		slog.Error("postgres migrate", "err", err)
		os.Exit(1)
	}
	slog.Info("postgres ready")

	// Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("redis URL parse", "err", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		slog.Error("redis ping", "err", err)
		os.Exit(1)
	}
	slog.Info("redis ready")

	// AI + scraping (existing services).
	aiService, err := services.NewAIService()
	if err != nil {
		slog.Error("AI service init failed — set MISTRAL_API_KEY (or another provider key) in .env", "err", err)
		os.Exit(1)
	}
	slog.Info("ai provider ready", "provider", aiService.ProviderName())
	scraper := services.NewScraperService()
	factChecker := services.NewFactCheckService(aiService, scraper)
	h := handlers.NewHandler(factChecker)

	// Auth
	captcha := auth.NewCaptcha(cfg.HCaptchaSecret)
	mailer := auth.LogMailer{}
	authSvc := auth.NewService(pool, captcha, mailer, cfg.BaseURL, cfg.CookieSecure)

	// Rate limiter
	limiter := ratelimit.New(rdb, ratelimit.Default())

	// Router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	// NOTE: chi's middleware.RealIP unconditionally trusts X-Forwarded-For,
	// which is the same spoof vector we just hardened against. Do NOT add
	// it — httpx.ClientIP handles the trusted-proxies check correctly.
	r.Use(slogRequestLogger())
	r.Use(middleware.Recoverer)
	// NO middleware.Timeout — it would kill /api/check/{id}/stream (SSE).
	// Per-route timeouts can be added on individual non-streaming endpoints later.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Every route gets the optional session attached.
	r.Use(auth.Optional(pool))
	// CSRF middleware enforces double-submit on mutating, authenticated requests.
	r.Use(auth.CSRF())

	// Public — no rate limit (cheap).
	r.Get("/health", h.HealthCheck)
	r.Get("/version", versionHandler)

	// Auth endpoints — rate-limited per IP.
	r.With(limiter.Middleware("/auth/signup")).Post("/auth/signup", authSvc.Signup)
	r.With(limiter.Middleware("/auth/login")).Post("/auth/login", authSvc.Login)
	r.With(limiter.Middleware("/auth/forgot")).Post("/auth/forgot", authSvc.Forgot)
	r.With(limiter.Middleware("/auth/reset")).Post("/auth/reset", authSvc.Reset)
	r.With(limiter.Middleware("/auth/verify")).Post("/auth/verify", authSvc.Verify)
	r.With(limiter.Middleware("/auth/logout")).Post("/auth/logout", authSvc.Logout)
	r.Get("/auth/me", authSvc.Me)

	// Fact-check — rate-limited per (plan, scope).
	r.With(limiter.Middleware("/api/check")).Post("/api/check", func(w http.ResponseWriter, req *http.Request) {
		// Audit who initiated the check (anon checks log with NULL user).
		var uid *int64
		if sess, ok := auth.FromContext(req.Context()); ok {
			uid = &sess.UserID
		}
		audit.Log(req.Context(), pool, uid, "check.started", "check", "", req, nil)
		h.CheckFacts(w, req)
	})
	r.Get("/api/check/{id}", h.GetCheckStatus)
	r.Get("/api/check/{id}/stream", h.StreamCheckProgress)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second, // slowloris on headers
		ReadTimeout:       30 * time.Second, // slowloris on body — auth bodies are small
		// WriteTimeout intentionally 0 — /api/check/{id}/stream is long-lived SSE.
		// IdleTimeout guards keepalive connections.
		IdleTimeout: 120 * time.Second,
	}

	go func() {
		slog.Info("Alethea API listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", "err", err)
			stop()
		}
	}()

	<-rootCtx.Done()
	slog.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"name":"alethea-api","phase":"1"}`))
}

// slogRequestLogger is a tiny replacement for chi's middleware.Logger so we
// can emit structured access logs through slog instead of stdlib log.
func slogRequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			slog.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", httpx.ClientIP(r),
				"req_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
