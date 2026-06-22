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
	"alethea/api/byok"
	"alethea/api/config"
	"alethea/api/db"
	"alethea/api/handlers"
	"alethea/api/httpx"
	alog "alethea/api/log"
	"alethea/api/ratelimit"
	"alethea/api/services"
	"alethea/api/services/ai"
	"alethea/api/services/byokresolver"
	"alethea/api/services/factcheck"
	"alethea/api/services/factcheck/cache"
	"alethea/api/services/factcheck/checkstream"
	"alethea/api/services/search"
	"alethea/api/services/sources"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("dotenv parse error: %v", err)
		os.Exit(1)
	}

	alog.Init(os.Getenv("DEBUG") == "true")

	if err := config.Load(); err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	cfg := config.App

	prefixes, err := httpx.ParseProxyCIDRs(cfg.TrustedProxies)
	if err != nil {
		slog.Warn("TRUSTED_PROXIES contained bad entries — those entries ignored", "err", err)
	}
	httpx.SetTrustedProxies(prefixes)
	if len(prefixes) == 0 {
		slog.Info("TRUSTED_PROXIES is empty — X-Forwarded-For will be ignored")
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

	// AI providers — pooled. Strong = Mistral large (judge + investigators);
	// Screener also Mistral but on whatever the env default is.
	strongProv, err := ai.NewDefaultProvider()
	if err != nil {
		slog.Error("AI provider init failed — set MISTRAL_API_KEY (or another provider key) in .env", "err", err)
		os.Exit(1)
	}
	slog.Info("ai provider ready", "provider", strongProv.Name(), "model", strongProv.ModelID())

	// Legacy services still used for: scraper + image analysis through the
	// existing aiservice wrapper. Pipeline calls the scraper directly.
	scraperSvc := services.NewScraperService()

	// BYOK vault. Missing master key is non-fatal — BYOK endpoints will 503
	// with a clear message; the pipeline runs pool-only.
	var vault *byok.Vault
	if cfg.BYOKMasterKey != "" {
		vault, err = byok.New(cfg.BYOKMasterKey)
		if err != nil {
			slog.Error("BYOK init failed — fix BYOK_MASTER_KEY or unset it", "err", err)
			os.Exit(1)
		}
		slog.Info("BYOK vault ready")
	} else {
		slog.Warn("BYOK disabled — BYOK_MASTER_KEY not set (BYOK endpoints will 503)")
	}

	// Search providers. Brave primary, Tavily fallback. Empty key ⇒ provider absent.
	var searchProviders []search.Provider
	if k := os.Getenv("BRAVE_SEARCH_API_KEY"); k != "" {
		searchProviders = append(searchProviders, search.NewBrave(k))
	}
	if k := os.Getenv("TAVILY_API_KEY"); k != "" {
		searchProviders = append(searchProviders, search.NewTavily(k))
	}
	if len(searchProviders) == 0 {
		slog.Warn("no search provider configured — pipeline will return unverifiable verdicts. Set BRAVE_SEARCH_API_KEY or TAVILY_API_KEY.")
	}
	sx := search.NewMulti(rdb, 10*time.Minute, searchProviders...)

	// Curated source registry — loaded eagerly so the first check doesn't pay
	// the load latency, and refreshed every 5 min thereafter.
	sourceReg := sources.New(pool)
	if err := sourceReg.Reload(rootCtx); err != nil {
		slog.Warn("source registry initial load failed; will retry lazily", "err", err)
	}

	// Pipeline.
	resolver := byokresolver.New(byokresolver.Pool{Screener: strongProv, Strong: strongProv}, vault, pool)
	verdictCache := cache.New(rdb, factcheck.PromptVersion)
	hub := checkstream.NewHub()
	pipeline := factcheck.NewPipeline(factcheck.Deps{
		DB: pool, Resolver: resolver, Search: sx, Cache: verdictCache, Hub: hub,
		Scraper:  scraperSvc,
		Reranker: factcheck.SourceReranker{Registry: sourceReg},
	})
	checkH := handlers.NewCheckV2(pool, pipeline, hub)
	byokH := handlers.NewBYOKKeys(pool, vault)
	sourcesH := handlers.NewSources(pool, sourceReg)
	journalistH := handlers.NewJournalist(pool, sourceReg)

	// Legacy single-shot handler kept ONLY for /health; the /api/check
	// endpoints are now backed by the new pipeline.
	legacyH := handlers.NewHandler(nil)

	// Auth
	captcha := auth.NewCaptcha(cfg.HCaptchaSecret)
	mailer := auth.LogMailer{}
	authSvc := auth.NewService(pool, captcha, mailer, cfg.BaseURL, cfg.CookieSecure)

	// Rate limiter
	limiter := ratelimit.New(rdb, ratelimit.Default())

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(slogRequestLogger())
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(auth.Optional(pool))
	r.Use(auth.CSRF())

	// Public
	r.Get("/health", legacyH.HealthCheck)
	r.Get("/version", versionHandler)

	// Auth
	r.With(limiter.Middleware("/auth/signup")).Post("/auth/signup", authSvc.Signup)
	r.With(limiter.Middleware("/auth/login")).Post("/auth/login", authSvc.Login)
	r.With(limiter.Middleware("/auth/forgot")).Post("/auth/forgot", authSvc.Forgot)
	r.With(limiter.Middleware("/auth/reset")).Post("/auth/reset", authSvc.Reset)
	r.With(limiter.Middleware("/auth/verify")).Post("/auth/verify", authSvc.Verify)
	r.With(limiter.Middleware("/auth/logout")).Post("/auth/logout", authSvc.Logout)
	r.Get("/auth/me", authSvc.Me)

	// Fact-check v2 (pipeline-backed)
	r.With(limiter.Middleware("/api/check")).Post("/api/check", func(w http.ResponseWriter, req *http.Request) {
		var uid *int64
		if sess, ok := auth.FromContext(req.Context()); ok {
			uid = &sess.UserID
		}
		audit.Log(req.Context(), pool, uid, "check.started", "check", "", req, nil)
		checkH.Start(w, req)
	})
	r.With(limiter.Middleware("/api/check/get")).Get("/api/check/{id}", checkH.Get)
	r.With(limiter.Middleware("/api/check/stream")).Get("/api/check/{id}/stream", checkH.Stream)

	// BYOK key management (signed-in only, rate-limited per user)
	r.With(auth.Required(pool), limiter.Middleware("/api/me/keys")).Get("/api/me/keys", byokH.List)
	r.With(auth.Required(pool), limiter.Middleware("/api/me/keys")).Post("/api/me/keys", byokH.Set)
	r.With(auth.Required(pool), limiter.Middleware("/api/me/keys")).Delete("/api/me/keys", byokH.Delete)

	// Curated sources — public list, admin CRUD
	r.With(limiter.Middleware("/api/sources")).Get("/api/sources", sourcesH.List)
	r.With(auth.Required(pool), limiter.Middleware("/api/admin/sources")).Post("/api/admin/sources", sourcesH.Add)
	r.With(auth.Required(pool), limiter.Middleware("/api/admin/sources")).Delete("/api/admin/sources", sourcesH.Deactivate)

	// Journalist program
	r.With(auth.Required(pool), limiter.Middleware("/api/me/journalist")).Post("/api/me/journalist-application", journalistH.Submit)
	r.With(auth.Required(pool), limiter.Middleware("/api/me/journalist")).Get("/api/me/journalist-application", journalistH.Mine)
	r.With(auth.Required(pool), limiter.Middleware("/api/me/journalist")).Delete("/api/me/journalist-application", journalistH.Withdraw)
	r.With(auth.Required(pool), limiter.Middleware("/api/admin/journalist")).Get("/api/admin/journalist-applications", journalistH.AdminList)
	r.With(auth.Required(pool), limiter.Middleware("/api/admin/journalist")).Post("/api/admin/journalist-applications/{id}/decide", journalistH.AdminDecide)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
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
	_, _ = w.Write([]byte(`{"name":"alethea-api","phase":"2"}`))
}

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
